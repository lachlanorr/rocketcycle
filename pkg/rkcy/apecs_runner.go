// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	otel_codes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

var (
	gInstanceCache *InstanceCache = NewInstanceCache()
)

func produceApecsTxnError(
	span trace.Span,
	rtxn *rtApecsTxn,
	step *ApecsTxn_Step,
	aprod *ApecsProducer,
	code Code,
	logToResult bool,
	format string,
	args ...interface{},
) {
	logMsg := fmt.Sprintf(format, args...)

	log.Error().
		Str("TraceId", rtxn.txn.TraceId).
		Msg(logMsg)

	span.SetStatus(otel_codes.Error, logMsg)
	if logToResult {
		span.RecordError(errors.New(logMsg))
	}

	err := aprod.produceError(rtxn, step, code, logToResult, logMsg)
	if err != nil {
		log.Error().
			Err(err).
			Str("TraceId", rtxn.txn.TraceId).
			Msg("produceApecsTxnError: Failed to produce error message")
	}
}

func produceNextStep(
	span trace.Span,
	rtxn *rtApecsTxn,
	step *ApecsTxn_Step,
	offset *Offset,
	aprod *ApecsProducer,
) {
	if rtxn.advanceStepIdx() {
		nextStep := rtxn.currentStep()
		// payload from last step should be passed to next step
		if nextStep.Payload == nil {
			nextStep.Payload = step.Result.Payload
		}
		if nextStep.System == System_STORAGE && nextStep.Offset == nil {
			// STORAGE steps always need the value PROCESS Offset
			if step.Offset != nil {
				// if current step has an offset, set that one
				// This allows multiple STORAGE steps in a row, and the same
				// PROCESS offset will get set on all of them.
				nextStep.Offset = step.Offset
			} else {
				// step has no offset, it's likely a PROCESS step, and we
				// default to the argument, which is the most recent offset
				// read from kafka
				nextStep.Offset = offset
			}
		}
		err := aprod.produceCurrentStep(rtxn.txn, rtxn.traceParent)
		if err != nil {
			produceApecsTxnError(span, rtxn, nextStep, aprod, Code_INTERNAL, true, "produceNextStep error=\"%s\": Failed produceCurrentStep for next step", err.Error())
			return
		}
	} else {
		// search for instance updates and create new storage txn to update storage
		if rtxn.txn.Direction == Direction_FORWARD {
			storageStepsMap := make(map[string]*ApecsTxn_Step)
			for _, step := range rtxn.txn.ForwardSteps {
				if step.System == System_PROCESS && step.Result.Instance != nil {
					stepKey := fmt.Sprintf("%s_%s", step.Concern, step.Key)
					storageStepsMap[stepKey] = &ApecsTxn_Step{
						System:  System_STORAGE,
						Concern: step.Concern,
						Command: UPDATE,
						Key:     step.Key,
						Payload: step.Result.Instance,
						Offset:  step.Offset,
					}
				}
			}
			if len(storageStepsMap) > 0 {
				storageSteps := make([]*ApecsTxn_Step, len(storageStepsMap))
				i := 0
				for _, v := range storageStepsMap {
					storageSteps[i] = v
					i++
				}
				storageTraceId := NewTraceId()
				rtxn.txn.AssocTraceId = storageTraceId
				err := aprod.executeTxn(
					storageTraceId,
					rtxn.txn.TraceId,
					rtxn.traceParent,
					false,
					storageSteps,
				)
				if err != nil {
					produceApecsTxnError(span, rtxn, step, aprod, Code_INTERNAL, true, "produceNextStep error=\"%s\" Key=%s: Failed to update storage", err.Error(), step.Key)
					return
				}
			}
		}

		err := aprod.produceComplete(rtxn)
		if err != nil {
			produceApecsTxnError(span, rtxn, step, aprod, Code_INTERNAL, true, "produceNextStep error=\"%s\": Failed to produceComplete", err.Error())
			return
		}
	}
}

func advanceApecsTxn(
	ctx context.Context,
	rtxn *rtApecsTxn,
	tp *TopicParts,
	offset *Offset,
	aprod *ApecsProducer,
) {
	ctx = InjectTraceParent(ctx, rtxn.traceParent)
	ctx, span, step := gPlatformImpl.Telem.StartStep(ctx, rtxn)
	defer span.End()

	log.Debug().
		Str("TraceId", rtxn.txn.TraceId).
		Msgf("Advancing ApecsTxn: %s %d %+v", Direction_name[int32(rtxn.txn.Direction)], rtxn.txn.CurrentStepIdx, step)

	if step.Result != nil {
		produceApecsTxnError(span, rtxn, step, aprod, Code_INTERNAL, true, "advanceApecsTxn Result=%+v: Current step already has Result", step.Result)
		return
	}

	if step.Concern != tp.Concern {
		produceApecsTxnError(span, rtxn, step, aprod, Code_INTERNAL, true, "advanceApecsTxn: Mismatched concern, expected=%s actual=%s", tp.Concern, step.Concern)
		return
	}

	if step.System != tp.System {
		produceApecsTxnError(span, rtxn, step, aprod, Code_INTERNAL, true, "advanceApecsTxn: Mismatched system, expected=%d actual=%d", tp.System, step.System)
		return
	}

	if step.Key == "" {
		produceApecsTxnError(span, rtxn, step, aprod, Code_INTERNAL, true, "advanceApecsTxn: No key in step")
		return
	}

	// Grab current timestamp, which will be used in a couple places below
	now := timestamppb.Now()

	// Read instance from InstanceCache
	var inst []byte
	if step.System == System_PROCESS {
		step.Offset = offset

		// Special case "REFRESH" command
		// REFRESH command is only ever sent after a READ was executed
		// against the Storage
		if step.Command == REFRESH {
			gInstanceCache.Set(step.Key, step.Payload, offset)
			step.Result = &ApecsTxn_Step_Result{
				Code:          Code_OK,
				ProcessedTime: now,
				EffectiveTime: now,
				Payload:       step.Payload,
			}
			produceNextStep(span, rtxn, step, offset, aprod)
			return
		} else {
			inst = gInstanceCache.Get(step.Key)

			nilOk := step.Command == CREATE || step.Command == VALIDATE_CREATE
			if inst == nil && !nilOk {
				// We should attempt to get the value from the DB, and we
				// do this by inserting a Storage READ step before this
				// one and sending things through again
				err := rtxn.insertSteps(
					rtxn.txn.CurrentStepIdx,
					&ApecsTxn_Step{
						System:  System_STORAGE,
						Concern: step.Concern,
						Command: READ,
						Key:     step.Key,
						Offset:  offset, // provide our PROCESS offset to the STORAGE step so it is recorded in the DB
					},
					&ApecsTxn_Step{
						System:  System_PROCESS,
						Concern: step.Concern,
						Command: REFRESH,
						Key:     step.Key,
					},
				)
				if err != nil {
					log.Error().Err(err).Msg("error in insertSteps")
					produceApecsTxnError(span, rtxn, nil, aprod, Code_INTERNAL, true, "advanceApecsTxn Key=%s: Unable to insert Storage READ implicit steps", step.Key)
					return
				}
				err = aprod.produceCurrentStep(rtxn.txn, rtxn.traceParent)
				if err != nil {
					produceApecsTxnError(span, rtxn, nil, aprod, Code_INTERNAL, true, "advanceApecsTxn error=\"%s\": Failed produceCurrentStep for next step", err.Error())
					return
				}
				return
			} else if inst != nil && step.Command == CREATE {
				produceApecsTxnError(span, rtxn, step, aprod, Code_INTERNAL, true, "advanceApecsTxn Key=%s: Instance already exists in cache in CREATE command", step.Key)
				return
			}
		}
	}

	if step.Offset == nil {
		produceApecsTxnError(span, rtxn, step, aprod, Code_INTERNAL, true, "advanceApecsTxn: Nil offset")
		return
	}

	args := &StepArgs{
		TraceId:       rtxn.txn.TraceId,
		ProcessedTime: now,
		Key:           step.Key,
		Instance:      inst,
		Payload:       step.Payload,
		Offset:        step.Offset,
	}

	step.Result = handleCommand(ctx, step.Concern, step.System, step.Command, rtxn.txn.Direction, args)

	if step.Result == nil {
		produceApecsTxnError(span, rtxn, step, aprod, Code_INTERNAL, true, "advanceApecsTxn Key=%s: nil result from step handler", step.Key)
		return
	}

	step.Result.ProcessedTime = now
	if step.Result.EffectiveTime == nil {
		step.Result.EffectiveTime = now
	}

	if step.Result.LogEvents != nil {
		for _, logEvt := range step.Result.LogEvents {
			sevMsg := fmt.Sprintf("%s %s", Severity_name[int32(logEvt.Sev)], logEvt.Msg)
			if logEvt.Sev == Severity_ERR {
				span.RecordError(errors.New(sevMsg))
			} else {
				span.AddEvent(sevMsg)
			}
		}
	}
	span.SetAttributes(
		attribute.String("rkcy.code", strconv.Itoa(int(step.Result.Code))),
	)
	if step.Result.Code != Code_OK {
		produceApecsTxnError(span, rtxn, step, aprod, step.Result.Code, false, "advanceApecsTxn Key=%s: Step failed with non OK result", step.Key)
		return
	}

	// Unless explcitly set by the handler, always assume we should
	// pass step payload in result to next step
	if step.Result.Payload == nil {
		step.Result.Payload = step.Payload
	}

	// Update InstanceCache if instance contents have changed
	if tp.System == System_PROCESS && step.Result.Instance != nil {
		gInstanceCache.Set(step.Key, step.Result.Instance, offset)
	}

	produceNextStep(span, rtxn, step, offset, aprod)
}

func consumeApecsTopic(
	ctx context.Context,
	adminBrokers string,
	consumerBrokers string,
	platformName string,
	fullTopic string,
	partition int32,
	tp *TopicParts,
) {
	aprod := NewApecsProducer(ctx, adminBrokers, platformName, nil)

	groupName := fmt.Sprintf("rkcy_%s", fullTopic)
	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        consumerBrokers,
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
				if directive == Directive_APECS_TXN {
					txn := ApecsTxn{}
					err := proto.Unmarshal(msg.Value, &txn)
					if err != nil {
						log.Error().
							Err(err).
							Msg("Failed to Unmarshal ApecsTxn")
					} else {
						offset := &Offset{
							Generation: tp.Generation,
							Partition:  partition,
							Offset:     int64(msg.TopicPartition.Offset),
						}

						rtxn, err := newRtApecsTxn(&txn, GetTraceParent(msg))
						if err != nil {
							log.Error().
								Err(err).
								Msg("Failed to create RtApecsTxn")
						} else {
							advanceApecsTxn(ctx, rtxn, tp, offset, aprod)
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

func startApecsRunner(
	ctx context.Context,
	adminBrokers string,
	consumerBrokers string,
	platformName string,
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

	if tp.ConcernType != Platform_Concern_APECS {
		log.Fatal().
			Err(err).
			Str("Topic", fullTopic).
			Msg("startApecsRunner: not an APECS topic")
	}

	go consumeApecsTopic(
		ctx,
		adminBrokers,
		consumerBrokers,
		platformName,
		fullTopic,
		partition,
		tp,
	)
}
