// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
)

var (
	instanceCache *InstanceCache = NewInstanceCache()
)

func produceApecsTxnError(
	rtxn *rtApecsTxn,
	step *pb.Step,
	aprod *ApecsProducer,
	code pb.Code,
	logToResult bool,
	format string,
	args ...interface{},
) {
	logMsg := fmt.Sprintf(format, args...)

	log.Error().
		Str("ReqId", rtxn.txn.ReqId).
		Msg(logMsg)

	err := aprod.produceError(rtxn, step, code, logToResult, logMsg)
	if err != nil {
		log.Error().
			Err(err).
			Str("ReqId", rtxn.txn.ReqId).
			Msg("produceApecsTxnError: Failed to produce error message")
	}
}

func produceNextStep(
	rtxn *rtApecsTxn,
	step *pb.Step,
	rsltPayload []byte,
	aprod *ApecsProducer,
) {
	if rtxn.advanceStepIdx() {
		nextStep := rtxn.currentStep()
		// payload from last step should be passed to next step
		nextStep.Payload = rsltPayload
		err := aprod.produceCurrentStep(rtxn.txn)
		if err != nil {
			produceApecsTxnError(rtxn, nextStep, aprod, pb.Code_INTERNAL, true, "advanceApecsTxn error=\"%s\": Failed produceCurrentStep for next step", err.Error())
			return
		}
	} else {
		err := aprod.produceComplete(rtxn)
		if err != nil {
			produceApecsTxnError(rtxn, step, aprod, pb.Code_INTERNAL, true, "advanceApecsTxn error=\"%s\": Failed to produceComplete", err.Error())
			return
		}
	}
}

func advanceApecsTxn(
	ctx context.Context,
	rtxn *rtApecsTxn,
	tp *TopicParts,
	offset *pb.Offset,
	handlers map[pb.Command]Handler,
	aprod *ApecsProducer,
) {
	step := rtxn.currentStep()

	if step.Result != nil {
		produceApecsTxnError(rtxn, step, aprod, pb.Code_INTERNAL, true, "advanceApecsTxn Result=%+v: Current step already has Result", step.Result)
		return
	}

	if step.ConcernName != tp.ConcernName {
		produceApecsTxnError(rtxn, step, aprod, pb.Code_INTERNAL, true, "advanceApecsTxn: Mismatched concern, expected=%s actual=%s", tp.ConcernName, step.ConcernName)
		return
	}

	if step.Key == "" {
		produceApecsTxnError(rtxn, step, aprod, pb.Code_INTERNAL, true, "advanceApecsTxn: No key in step")
		return
	}

	// Grab current timestamp, which will be used in a couple places below
	now := timestamppb.Now()

	// Read instance from InstanceCache
	var inst []byte
	if tp.System == pb.System_PROCESS {
		// Special case "REFRESH" command
		// REFRESH command is only ever sent after a READ was executed
		// against the Storage
		if step.Command == pb.Command_REFRESH {
			instanceCache.Set(step.Key, step.Payload)
			step.Result = &pb.Step_Result{
				Code:          pb.Code_OK,
				ProcessedTime: now,
				EffectiveTime: now,
			}
			produceNextStep(rtxn, step, nil, aprod)
			return
		} else {
			inst = instanceCache.Get(step.Key)

			expectingNil := step.Command == pb.Command_CREATE || step.Command == pb.Command_VALIDATE
			if inst == nil && !expectingNil {
				// We should attempt to get the value from the DB, and we
				// do this by inserting a Storage READ step before this
				// one and sending things through again
				err := rtxn.insertSteps(
					rtxn.txn.CurrentStepIdx,
					&pb.Step{
						System:      pb.System_STORAGE,
						ConcernName: step.ConcernName,
						Command:     pb.Command_READ,
						Key:         step.Key,
					},
					&pb.Step{
						System:      pb.System_PROCESS,
						ConcernName: step.ConcernName,
						Command:     pb.Command_REFRESH,
						Key:         step.Key,
					},
				)
				if err != nil {
					log.Error().Err(err).Msg("error in insertSteps")
					produceApecsTxnError(rtxn, nil, aprod, pb.Code_INTERNAL, true, "advanceApecsTxn Key=%s: Unable to insert Storage READ implicit steps", step.Key)
					return
				}
				err = aprod.produceCurrentStep(rtxn.txn)
				if err != nil {
					produceApecsTxnError(rtxn, nil, aprod, pb.Code_INTERNAL, true, "advanceApecsTxn error=\"%s\": Failed produceCurrentStep for next step", err.Error())
					return
				}
				return
			} else if inst != nil && expectingNil {
				produceApecsTxnError(rtxn, step, aprod, pb.Code_INTERNAL, true, "advanceApecsTxn Key=%s: Instance already exists in cache in CREATE/VALIDATE command", step.Key)
				return
			}
		}
	}

	hndlr, ok := handlers[step.Command]
	if !ok {
		produceApecsTxnError(rtxn, step, aprod, pb.Code_UNKNOWN_COMMAND, true, "advanceApecsTxn Command=%d: No handler for command", step.Command)
		return
	}
	if rtxn.txn.Direction == pb.Direction_FORWARD && hndlr.Do == nil {
		produceApecsTxnError(rtxn, step, aprod, pb.Code_UNKNOWN_COMMAND, true, "advanceApecsTxn Command=%d: No Do handler function for command", step.Command)
		return
	}
	if rtxn.txn.Direction == pb.Direction_REVERSE && hndlr.Undo == nil {
		produceApecsTxnError(rtxn, step, aprod, pb.Code_UNKNOWN_COMMAND, true, "advanceApecsTxn Command=%d: No Undo handler function for command", step.Command)
		return
	}

	args := StepArgs{
		ReqId:         rtxn.txn.ReqId,
		ProcessedTime: now,
		Key:           step.Key,
		Instance:      inst,
		Payload:       step.Payload,
		Offset:        offset,
	}

	var rslt *StepResult
	if rtxn.txn.Direction == pb.Direction_FORWARD {
		rslt = hndlr.Do(ctx, &args)
	} else {
		rslt = hndlr.Undo(ctx, &args)
	}

	if rslt == nil {
		produceApecsTxnError(rtxn, step, aprod, pb.Code_INTERNAL, true, "advanceApecsTxn Key=%s: nil result from step handler", step.Key)
		return
	}

	effectiveTime := now
	// if handler set EffectiveTime in result, we pick up the change here
	if rslt.EffectiveTime != nil {
		effectiveTime = rslt.EffectiveTime
	}

	step.Result = &pb.Step_Result{
		Code:          rslt.Code,
		ProcessedTime: now,
		EffectiveTime: effectiveTime,
		LogEvents:     rslt.LogEvents,
		Payload:       rslt.Payload,
	}

	if step.Result.Code != pb.Code_OK {
		produceApecsTxnError(rtxn, step, aprod, step.Result.Code, false, "advanceApecsTxn Key=%s: Step failed with non OK result", step.Key)
		return
	}

	if tp.System == pb.System_PROCESS {
		if rslt.Instance != nil {
			// Instance has changed in handler, update the
			// storage system
			err := aprod.executeTxn(
				uuid.NewString(),
				nil,
				false,
				[]pb.Step{
					{
						System:      pb.System_STORAGE,
						ConcernName: step.ConcernName,
						Command:     pb.Command_UPDATE,
						Key:         step.Key,
						Payload:     rslt.Instance,
					},
				},
			)
			if err != nil {
				produceApecsTxnError(rtxn, step, aprod, pb.Code_INTERNAL, true, "advanceApecsTxn error=\"%s\" Key=%s: Failed to update storage", err.Error(), step.Key)
				return
			}
			instanceCache.Set(step.Key, rslt.Instance)
		}
	}

	produceNextStep(rtxn, step, rslt.Payload, aprod)
}

func consumeApecsTopic(
	ctx context.Context,
	clusterBootstrap string,
	fullTopic string,
	partition int32,
	tp *TopicParts,
	handlers map[pb.Command]Handler,
) {
	aprod := NewApecsProducer(ctx, settings.BootstrapServers, platformName)

	groupName := fmt.Sprintf("rkcy_%s", fullTopic)
	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        clusterBootstrap,
		"group.id":                 groupName,
		"enable.auto.commit":       true,  // librdkafka will commit to brokers for us on an interval and when we close consumer
		"enable.auto.offset.store": false, // we explicitely commit to local store to get "at least once" behavior
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("failed to kafka.NewConsumer")
	}
	defer cons.Close()

	err = cons.Assign([]kafka.TopicPartition{
		{
			Topic:     &fullTopic,
			Partition: partition,
			Offset:    kafka.OffsetStored,
		},
	})

	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to Assign")
	}

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("consumeStorage exiting, ctx.Done()")
			return
		default:
			msg, err := cons.ReadMessage(time.Second * 5)
			timedOut := err != nil && err.(kafka.Error).Code() == kafka.ErrTimedOut
			if err != nil && !timedOut {
				log.Fatal().
					Err(err).
					Msg("Error during ReadMessage")
			}

			if !timedOut && msg != nil {
				directive := GetDirective(msg)
				if directive == pb.Directive_APECS_TXN {
					txn := pb.ApecsTxn{}
					err := proto.Unmarshal(msg.Value, &txn)
					if err != nil {
						log.Error().
							Err(err).
							Msg("Failed to Unmarshal ApecsTxn")
					} else {
						offset := &pb.Offset{
							Generation: tp.Generation,
							Partition:  partition,
							Offset:     int64(msg.TopicPartition.Offset),
						}

						rtxn, err := newRtApecsTxn(&txn)
						if err != nil {
							log.Error().
								Err(err).
								Msg("Failed to create RtApecsTxn")
						} else {
							advanceApecsTxn(ctx, rtxn, tp, offset, handlers, aprod)
						}
					}
				} else {
					log.Warn().
						Int("Directive", int(directive)).
						Msg("Invalid directive on ApecsTxn topic")
				}

				_, err = cons.StoreOffsets([]kafka.TopicPartition{
					{
						Topic:     &fullTopic,
						Partition: partition,
						Offset:    msg.TopicPartition.Offset + 1,
					},
				})
				if err != nil {
					log.Fatal().
						Err(err).
						Msgf("Unable to store offsets %s/%d/%d", fullTopic, partition, msg.TopicPartition.Offset)
				}
			}
		}
	}
}

type PlatformHandlers map[string]map[pb.System]map[pb.Command]Handler

func processHandlerCreate(ctx context.Context, stepInfo *StepArgs) *StepResult {
	instanceCache.Set(stepInfo.Key, stepInfo.Payload)
	return &StepResult{
		Code:    pb.Code_OK,
		Payload: stepInfo.Payload,
	}
}

func processHandlerRead(ctx context.Context, stepInfo *StepArgs) *StepResult {
	return &StepResult{
		Code:    pb.Code_OK,
		Payload: stepInfo.Instance,
	}
}

func registerProcessCrudHandlers(handlers map[pb.Command]Handler) {
	_, ok := handlers[pb.Command_REFRESH]
	if ok {
		// Command_REFRESH is always handled explicitly in advanceApecsTxn
		log.Warn().
			Msg("Command_REFRESH should never be specified, ignoring")
	}

	_, ok = handlers[pb.Command_CREATE]
	if ok {
		log.Warn().
			Msg("Overriding Command_CREATE handler")
	}
	handlers[pb.Command_CREATE] = Handler{
		Do: processHandlerCreate,
	}

	_, ok = handlers[pb.Command_READ]
	if ok {
		log.Warn().
			Msg("Overriding Command_READ handler")
	}
	handlers[pb.Command_READ] = Handler{
		Do: processHandlerRead,
	}
}

func startApecsRunner(
	ctx context.Context,
	platHndlrs PlatformHandlers,
	clusterBootstrap string,
	fullTopic string,
	partition int32,
) {
	tp, err := ParseFullTopicName(fullTopic)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("Topic", fullTopic).
			Msg("startApecsRunner: failed to ParseFullTopicName")
	}

	if tp.ConcernType != pb.Platform_Concern_APECS {
		log.Fatal().
			Err(err).
			Str("Topic", fullTopic).
			Msg("startApecsRunner: not an APECS topic")
	}

	concernHandlers, ok := platHndlrs[tp.ConcernName]
	if !ok {
		log.Fatal().
			Err(err).
			Str("Topic", fullTopic).
			Str("Concern", tp.ConcernName).
			Msg("startApecsRunner: no handlers for concern")
	}

	var handlers map[pb.Command]Handler
	if tp.System == pb.System_PROCESS {
		handlers = concernHandlers[pb.System_PROCESS]

		// Insert handlers for process CRUD ops
		registerProcessCrudHandlers(handlers)
	} else if tp.System == pb.System_STORAGE {
		handlers = concernHandlers[pb.System_STORAGE]
	} else {
		log.Fatal().
			Err(err).
			Str("Topic", fullTopic).
			Int("System", int(tp.System)).
			Msg("startApecsRunner: invalid system")
	}

	if handlers == nil || len(handlers) == 0 {
		log.Fatal().
			Str("Topic", fullTopic).
			Str("Concern", tp.ConcernName).
			Int("System", int(tp.System)).
			Msg("startApecsRunner: empty handlers")
	}

	go consumeApecsTopic(
		ctx,
		clusterBootstrap,
		fullTopic,
		partition,
		tp,
		handlers,
	)
}
