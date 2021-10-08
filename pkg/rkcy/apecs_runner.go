// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	otel_codes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

var (
	gInstanceCache *InstanceCache = NewInstanceCache()
)

func produceApecsTxnError(
	ctx context.Context,
	span trace.Span,
	rtxn *rtApecsTxn,
	step *ApecsTxn_Step,
	aprod *ApecsProducer,
	code Code,
	logToResult bool,
	wg *sync.WaitGroup,
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

	err := aprod.produceError(ctx, rtxn, step, code, logToResult, logMsg, wg)
	if err != nil {
		log.Error().
			Err(err).
			Str("TraceId", rtxn.txn.TraceId).
			Msg("produceApecsTxnError: Failed to produce error message")
	}
}

func produceNextStep(
	ctx context.Context,
	span trace.Span,
	rtxn *rtApecsTxn,
	step *ApecsTxn_Step,
	offset *Offset,
	aprod *ApecsProducer,
	wg *sync.WaitGroup,
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
		err := aprod.produceCurrentStep(rtxn.txn, rtxn.traceParent, wg)
		if err != nil {
			produceApecsTxnError(ctx, span, rtxn, nextStep, aprod, Code_INTERNAL, true, wg, "produceNextStep error=\"%s\": Failed produceCurrentStep for next step", err.Error())
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

			// generate a distinct storage transaction with a single
			// UPDATE storage step for every instance modified
			for _, step := range storageStepsMap {
				storageSteps := []*ApecsTxn_Step{step}
				storageTraceId := NewTraceId()
				rtxn.txn.AssocTraceIds = append(rtxn.txn.AssocTraceIds, storageTraceId)

				err := aprod.executeTxn(
					storageTraceId,
					rtxn.txn.TraceId,
					rtxn.traceParent,
					UponError_BAILOUT,
					storageSteps,
					wg,
				)
				if err != nil {
					produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "produceNextStep error=\"%s\" Key=%s: Failed to update storage", err.Error(), step.Key)
					// This should be extremely rare, maybe impossible
					// assuming we can produce at all.  Without proper
					// storage messages going through, we are better
					// of failing very loudly.
					log.Fatal().
						Msgf("produceNextStep error=\"%s\" Key=%s: Failed to update storage", err.Error(), step.Key)
				}
			}
		}

		err := aprod.produceComplete(ctx, rtxn, wg)
		if err != nil {
			produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "produceNextStep error=\"%s\": Failed to produceComplete", err.Error())
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
	wg *sync.WaitGroup,
) Code {
	ctx = InjectTraceParent(ctx, rtxn.traceParent)
	ctx, span, step := gPlatformImpl.Telem.StartStep(ctx, rtxn)
	defer span.End()

	log.Debug().
		Str("TraceId", rtxn.txn.TraceId).
		Msgf("Advancing ApecsTxn: %s %d %+v", Direction_name[int32(rtxn.txn.Direction)], rtxn.txn.CurrentStepIdx, step)

	if step.Result != nil {
		produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn Result=%+v: Current step already has Result", step.Result)
		return Code_INTERNAL
	}

	if step.Concern != tp.Concern {
		produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn: Mismatched concern, expected=%s actual=%s", tp.Concern, step.Concern)
		return Code_INTERNAL
	}

	if step.System != tp.System {
		produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn: Mismatched system, expected=%d actual=%d", tp.System, step.System)
		return Code_INTERNAL
	}

	if step.Key == "" {
		produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn: No key in step")
		return Code_INTERNAL
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
			produceNextStep(ctx, span, rtxn, step, offset, aprod, wg)
			return Code_OK
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
					produceApecsTxnError(ctx, span, rtxn, nil, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn Key=%s: Unable to insert Storage READ implicit steps", step.Key)
					return Code_INTERNAL
				}
				err = aprod.produceCurrentStep(rtxn.txn, rtxn.traceParent, wg)
				if err != nil {
					produceApecsTxnError(ctx, span, rtxn, nil, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn error=\"%s\": Failed produceCurrentStep for next step", err.Error())
					return Code_INTERNAL
				}
				return Code_OK
			} else if inst != nil && step.Command == CREATE {
				produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn Key=%s: Instance already exists in cache in CREATE command", step.Key)
				return Code_INTERNAL
			}
		}
	}

	if step.Offset == nil {
		produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn: Nil offset")
		return Code_INTERNAL
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

	if (step.Result == nil || step.Result.Code != Code_OK) &&
		rtxn.txn.UponError == UponError_BAILOUT {
		// We'll bailout up the chain, but down here we can at least
		// log any info we have about the result.
		var code Code
		if step.Result != nil {
			code = step.Result.Code
			for _, logEvt := range step.Result.LogEvents {
				log.Error().
					Str("TraceId", args.TraceId).
					Str("Concern", step.Concern).
					Str("System", step.System.String()).
					Str("Command", step.Command).
					Str("Direction", rtxn.txn.Direction.String()).
					Msgf("BAILOUT LogEvent: %s %s", Severity_name[int32(logEvt.Sev)], logEvt.Msg)
			}
		} else {
			code = Code_NIL_RESULT
		}
		return code
	}

	if step.Result == nil {
		produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn Key=%s: nil result from step handler", step.Key)
		return step.Result.Code
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
		produceApecsTxnError(ctx, span, rtxn, step, aprod, step.Result.Code, false, wg, "advanceApecsTxn Key=%s: Step failed with non OK result", step.Key)
		return step.Result.Code
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

	produceNextStep(ctx, span, rtxn, step, offset, aprod, wg)
	return step.Result.Code
}

func consumeApecsTopic(
	ctx context.Context,
	adminBrokers string,
	consumerBrokers string,
	platformName string,
	fullTopic string,
	partition int32,
	tp *TopicParts,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	aprod := NewApecsProducer(ctx, adminBrokers, platformName, nil, wg)

	groupName := fmt.Sprintf("rkcy_apecs_%s", fullTopic)
	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        consumerBrokers,
		"group.id":                 groupName,
		"enable.auto.commit":       false, // we commit manually on an interval
		"enable.auto.offset.store": false, // we commit to local store after processeing to get at least once behavior
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("failed to kafka.NewConsumer")
	}
	shouldCommit := false
	defer func() {
		log.Warn().
			Str("Topic", fullTopic).
			Msgf("Closing kafka consumer")
		if shouldCommit {
			_, err = cons.Commit()
			shouldCommit = false
			if err != nil {
				log.Error().
					Err(err).
					Msgf("Unable to commit")
			}
		}
		cons.Close()
		log.Warn().
			Str("Topic", fullTopic).
			Msgf("Closed kafka consumer")
	}()

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

	firstMessage := true
	commitTicker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			log.Warn().
				Msg("consumeStorage exiting, ctx.Done()")
			return
		case <-commitTicker.C:
			if shouldCommit {
				_, err = cons.Commit()
				shouldCommit = false
				if err != nil {
					log.Fatal().
						Err(err).
						Msgf("Unable to commit")
				}
			}
		default:
			msg, err := cons.ReadMessage(100 * time.Millisecond)
			timedOut := err != nil && err.(kafka.Error).Code() == kafka.ErrTimedOut
			if err != nil && !timedOut {
				log.Fatal().
					Err(err).
					Msg("Error during ReadMessage")
			} else if !timedOut && msg != nil {
				// If this is the first message read, commit the
				// offset to current to ensure we have an offset in
				// kafka, and if we blow up and start again we will
				// not default to latest.
				if firstMessage {
					firstMessage = false
					log.Info().
						Str("Topic", fullTopic).
						Int32("Partition", partition).
						Int64("Offset", int64(msg.TopicPartition.Offset)).
						Msg("Initial commit to current offset")

					comOffs, err := cons.CommitOffsets([]kafka.TopicPartition{
						{
							Topic:     &fullTopic,
							Partition: partition,
							Offset:    msg.TopicPartition.Offset,
						},
					})
					if err != nil {
						log.Fatal().
							Err(err).
							Str("Topic", fullTopic).
							Int32("Partition", partition).
							Int64("Offset", int64(msg.TopicPartition.Offset)).
							Msgf("Unable to commit initial offset")
					}
					log.Info().
						Str("Topic", fullTopic).
						Int32("Partition", partition).
						Int64("Offset", int64(msg.TopicPartition.Offset)).
						Msgf("Initial offset committed: %+v", comOffs)
				}

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
							code := advanceApecsTxn(ctx, rtxn, tp, offset, aprod, wg)
							// Bailout is necessary when a storage step fails.
							// Those must be retried indefinitely until success.
							// So... we force commit the offset to current and
							// kill ourselves, and we'll pick up where we left
							// off when we are restarted.
							if code != Code_OK && rtxn.txn.UponError == UponError_BAILOUT {
								_, err := cons.CommitOffsets([]kafka.TopicPartition{
									{
										Topic:     &fullTopic,
										Partition: partition,
										Offset:    msg.TopicPartition.Offset,
									},
								})
								if err != nil {
									log.Error().
										Err(err).
										Str("Topic", fullTopic).
										Int32("Partition", partition).
										Int64("Offset", int64(msg.TopicPartition.Offset)).
										Msgf("BAILOUT commit error")
								}

								log.Fatal().
									Int("Code", int(code)).
									Str("TraceId", rtxn.txn.TraceId).
									Str("Txn", protojson.Format(proto.Message(rtxn.txn))).
									Str("Topic", fullTopic).
									Int32("Partition", partition).
									Int64("Offset", int64(msg.TopicPartition.Offset)).
									Msgf("BAILOUT")
							}
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
						Msgf("Unable to store offsets %s/%d/%d", fullTopic, partition, msg.TopicPartition.Offset+1)
				}
				shouldCommit = true
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
	wg *sync.WaitGroup,
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

	wg.Add(1)
	go consumeApecsTopic(
		ctx,
		adminBrokers,
		consumerBrokers,
		platformName,
		fullTopic,
		partition,
		tp,
		wg,
	)
}
