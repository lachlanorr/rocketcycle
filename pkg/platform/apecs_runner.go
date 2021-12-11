// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
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

	"github.com/lachlanorr/rocketcycle/pkg/config"
	"github.com/lachlanorr/rocketcycle/pkg/consumer"
	"github.com/lachlanorr/rocketcycle/pkg/jsonutils"
	"github.com/lachlanorr/rocketcycle/pkg/producer"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

func produceApecsTxnError(
	ctx context.Context,
	aprod rkcy.ApecsProducer,
	span trace.Span,
	rtxn *rkcy.RtApecsTxn,
	step *rkcypb.ApecsTxn_Step,
	code rkcypb.Code,
	logToResult bool,
	wg *sync.WaitGroup,
	format string,
	args ...interface{},
) {
	logMsg := fmt.Sprintf(format, args...)

	log.Error().
		Str("TxnId", rtxn.Txn.Id).
		Msg(logMsg)

	span.SetStatus(otel_codes.Error, logMsg)
	if logToResult {
		span.RecordError(errors.New(logMsg))
	}

	err := producer.ProduceError(ctx, aprod, rtxn, step, code, logToResult, logMsg, wg)
	if err != nil {
		log.Error().
			Err(err).
			Str("TxnId", rtxn.Txn.Id).
			Msg("produceApecsTxnError: Failed to produce error message")
	}
}

func produceNextStep(
	ctx context.Context,
	plat rkcy.Platform,
	aprod rkcy.ApecsProducer,
	span trace.Span,
	rtxn *rkcy.RtApecsTxn,
	step *rkcypb.ApecsTxn_Step,
	cmpdOffset *rkcypb.CompoundOffset,
	wg *sync.WaitGroup,
) {
	if rtxn.AdvanceStepIdx() {
		nextStep := rtxn.CurrentStep()
		// payload from last step should be passed to next step
		if nextStep.Payload == nil && rkcy.IsPlatformCommand(nextStep.Command) {
			nextStep.Payload = step.Result.Payload
		}
		if nextStep.System == rkcypb.System_STORAGE && nextStep.CmpdOffset == nil {
			// STORAGE steps always need the value PROCESS Offset
			if step.CmpdOffset != nil {
				// if current step has an offset, set that one
				// This allows multiple STORAGE steps in a row, and the same
				// PROCESS offset will get set on all of them.
				nextStep.CmpdOffset = step.CmpdOffset
			} else {
				// step has no offset, it's likely a PROCESS step, and we
				// default to the argument, which is the most recent offset
				// read from kafka
				nextStep.CmpdOffset = cmpdOffset
			}
		}
		err := producer.ProduceCurrentStep(aprod, rtxn.Txn, rtxn.TraceParent, wg)
		if err != nil {
			produceApecsTxnError(ctx, aprod, span, rtxn, nextStep, rkcypb.Code_INTERNAL, true, wg, "produceNextStep error=\"%s\": Failed produceCurrentStep for next step", err.Error())
			return
		}
	} else {
		// search for instance updates and create new storage txn to update storage
		if rtxn.Txn.Direction == rkcypb.Direction_FORWARD {
			if plat.System() == rkcypb.System_PROCESS {
				for _, step := range rtxn.Txn.ForwardSteps {
					if step.System == rkcypb.System_STORAGE {
						// generate secondary storage steps
						if step.Command == rkcy.CREATE || step.Command == rkcy.UPDATE || step.Command == rkcy.DELETE {
							storageStep := &rkcypb.ApecsTxn_Step{
								System:     rkcypb.System_STORAGE_SCND,
								Concern:    step.Concern,
								Command:    step.Command,
								Key:        step.Result.Key,
								Payload:    step.Result.Payload,
								CmpdOffset: step.CmpdOffset,
							}
							storageSteps := []*rkcypb.ApecsTxn_Step{storageStep}
							storageTxnId := rkcy.NewTraceId()
							rtxn.Txn.AssocTxns = append(rtxn.Txn.AssocTxns, &rkcypb.AssocTxn{Id: storageTxnId, Type: rkcypb.AssocTxn_SECONDARY_STORAGE})

							err := producer.ExecuteTxn(
								aprod,
								storageTxnId,
								&rkcypb.AssocTxn{Id: rtxn.Txn.Id, Type: rkcypb.AssocTxn_PARENT},
								rtxn.TraceParent,
								rkcypb.UponError_BAILOUT,
								storageSteps,
								wg,
							)
							if err != nil {
								produceApecsTxnError(ctx, aprod, span, rtxn, step, rkcypb.Code_INTERNAL, true, wg, "produceNextStep error=\"%s\" Key=%s: Failed to apply secondary storage steps", err.Error(), step.Key)
								// This should be extremely rare, maybe impossible
								// assuming we can produce at all.  Without proper
								// storage messages going through, we are better
								// off failing very loudly.
								log.Fatal().
									Msgf("produceNextStep error=\"%s\" Key=%s: Failed to generate secondar storage steps", err.Error(), step.Key)
							}
						}
					}
				}
			}

			// Generate update steps for any changed instances
			storageStepsMap := make(map[string]*rkcypb.ApecsTxn_Step)
			for _, step := range rtxn.Txn.ForwardSteps {
				if step.System == rkcypb.System_PROCESS &&
					(step.Result.Instance != nil || step.Command == rkcy.REQUEST_RELATED || step.Command == rkcy.REFRESH_RELATED) {
					var stepInst []byte
					stepKey := fmt.Sprintf("%s_%s", step.Concern, step.Key)
					if step.Result.Instance != nil {
						stepInst = rkcy.PackPayloads(step.Result.Instance, plat.InstanceStore().GetRelated(step.Key))
					} else if step.Command == rkcy.REQUEST_RELATED {
						relRsp := &rkcypb.RelatedResponse{}
						err := proto.Unmarshal(step.Result.Payload, relRsp)
						if err != nil {
							log.Error().
								Err(err).
								Msgf("UPDATE_ASYNC ERROR: failed to decode RelatedResponse")
						} else {
							stepInst = relRsp.Payload
						}
					} else if step.Command == rkcy.REFRESH_RELATED {
						stepInst = step.Result.Payload
					}
					if stepInst == nil {
						log.Error().
							Msgf("UPDATE_ASYNC ERROR: Empty stepInst, stepKey=%s", stepKey)
					} else {
						storageStepsMap[stepKey] = &rkcypb.ApecsTxn_Step{
							System:        rkcypb.System_STORAGE,
							Concern:       step.Concern,
							Command:       rkcy.UPDATE_ASYNC,
							Key:           step.Key,
							Payload:       stepInst,
							CmpdOffset:    step.CmpdOffset,
							EffectiveTime: step.EffectiveTime,
						}
					}
				}
			}
			// generate a distinct storage transaction with a single
			// UPDATE_ASYNC storage step for every instance modified
			for _, step := range storageStepsMap {
				storageSteps := []*rkcypb.ApecsTxn_Step{step}
				storageTxnId := rkcy.NewTraceId()
				rtxn.Txn.AssocTxns = append(rtxn.Txn.AssocTxns, &rkcypb.AssocTxn{Id: storageTxnId, Type: rkcypb.AssocTxn_GENERATED_STORAGE})

				err := producer.ExecuteTxn(
					aprod,
					storageTxnId,
					&rkcypb.AssocTxn{Id: rtxn.Txn.Id, Type: rkcypb.AssocTxn_PARENT},
					rtxn.TraceParent,
					rkcypb.UponError_BAILOUT,
					storageSteps,
					wg,
				)
				if err != nil {
					produceApecsTxnError(ctx, aprod, span, rtxn, step, rkcypb.Code_INTERNAL, true, wg, "produceNextStep error=\"%s\" Key=%s: Failed to update storage", err.Error(), step.Key)
					// This should be extremely rare, maybe impossible
					// assuming we can produce at all.  Without proper
					// storage messages going through, we are better
					// off failing very loudly.
					log.Fatal().
						Msgf("produceNextStep error=\"%s\" Key=%s: Failed to update storage", err.Error(), step.Key)

				} else {
					// also execute secondary storage txn
					storageSteps[0].System = rkcypb.System_STORAGE_SCND
					storageScndTxnId := rkcy.NewTraceId()
					rtxn.Txn.AssocTxns = append(rtxn.Txn.AssocTxns, &rkcypb.AssocTxn{Id: storageTxnId, Type: rkcypb.AssocTxn_GENERATED_STORAGE})

					err := producer.ExecuteTxn(
						aprod,
						storageScndTxnId,
						&rkcypb.AssocTxn{Id: rtxn.Txn.Id, Type: rkcypb.AssocTxn_PARENT},
						rtxn.TraceParent,
						rkcypb.UponError_BAILOUT,
						storageSteps,
						wg,
					)
					if err != nil {
						produceApecsTxnError(ctx, aprod, span, rtxn, step, rkcypb.Code_INTERNAL, true, wg, "produceNextStep error=\"%s\" Key=%s: Failed to update secondary storage", err.Error(), step.Key)
						// This should be extremely rare, maybe impossible
						// assuming we can produce at all.  Without proper
						// storage messages going through, we are better
						// off failing very loudly.
						log.Fatal().
							Msgf("produceNextStep error=\"%s\" Key=%s: Failed to update secondary storage", err.Error(), step.Key)
					}
				}
			}
		}

		err := producer.ProduceComplete(ctx, aprod, rtxn, wg)
		if err != nil {
			produceApecsTxnError(ctx, aprod, span, rtxn, step, rkcypb.Code_INTERNAL, true, wg, "produceNextStep error=\"%s\": Failed to produceComplete", err.Error())
			return
		}
	}
}

func jsonNoErr(msg proto.Message) string {
	j, err := protojson.Marshal(msg)
	if err != nil {
		j = []byte(fmt.Sprintf(`{"err":%s}`, jsonutils.EncodeString(nil, err.Error())))
	}
	return string(j)
}

func cancelApecsTxn(
	ctx context.Context,
	plat rkcy.Platform,
	rtxn *rkcy.RtApecsTxn,
	tp *rkcy.TopicParts,
	cmpdOffset *rkcypb.CompoundOffset,
	aprod rkcy.ApecsProducer,
	confRdr rkcy.ConfigRdr,
	wg *sync.WaitGroup,
) {
	ctx = rkcy.InjectTraceParent(ctx, rtxn.TraceParent)
	ctx, span, step := plat.Telem().StartStep(ctx, rtxn)
	defer span.End()

	produceApecsTxnError(ctx, aprod, span, rtxn, step, rkcypb.Code_CANCELLED, true, wg, "Txn Cancelled")
}

func advanceApecsTxn(
	ctx context.Context,
	plat rkcy.Platform,
	rtxn *rkcy.RtApecsTxn,
	tp *rkcy.TopicParts,
	cmpdOffset *rkcypb.CompoundOffset,
	aprod rkcy.ApecsProducer,
	confRdr rkcy.ConfigRdr,
	wg *sync.WaitGroup,
) rkcypb.Code {
	ctx = rkcy.InjectTraceParent(ctx, rtxn.TraceParent)
	ctx, span, step := plat.Telem().StartStep(ctx, rtxn)
	defer span.End()

	log.Debug().
		Str("TxnId", rtxn.Txn.Id).
		Msgf("Advancing ApecsTxn: %s %d %s", rkcypb.Direction_name[int32(rtxn.Txn.Direction)], rtxn.Txn.CurrentStepIdx, jsonNoErr(step))

	if step.Result != nil {
		produceApecsTxnError(ctx, aprod, span, rtxn, step, rkcypb.Code_INTERNAL, true, wg, "advanceApecsTxn Result=%+v: Current step already has Result, %s", step.Result, jsonNoErr(step))
		return rkcypb.Code_INTERNAL
	}

	if step.Concern != tp.Concern {
		produceApecsTxnError(ctx, aprod, span, rtxn, step, rkcypb.Code_INTERNAL, true, wg, "advanceApecsTxn: Mismatched concern, expected=%s actual=%s, %s", tp.Concern, step.Concern, jsonNoErr(step))
		return rkcypb.Code_INTERNAL
	}

	if step.System != tp.System {
		produceApecsTxnError(ctx, aprod, span, rtxn, step, rkcypb.Code_INTERNAL, true, wg, "advanceApecsTxn: Mismatched system, expected=%d actual=%d, %s", tp.System, step.System, jsonNoErr(step))
		return rkcypb.Code_INTERNAL
	}

	// Special case, only storage CREATE and process VALIDATE_CREATE can have empty key
	if step.Key == "" && !rkcy.IsKeylessStep(step) {
		prevStep := rtxn.PreviousStep()
		if prevStep != nil && prevStep.Result != nil && prevStep.Result.Key != "" {
			step.Key = prevStep.Result.Key
		} else {
			produceApecsTxnError(ctx, aprod, span, rtxn, step, rkcypb.Code_INTERNAL, true, wg, "advanceApecsTxn: No key in step: %s %s, %s", step.System.String(), step.Command, jsonNoErr(step))
			return rkcypb.Code_INTERNAL
		}
	}

	// Grab current timestamp, which will be used in a couple places below
	now := timestamppb.Now()

	// Read instance from InstanceStore
	var inst []byte
	if step.System == rkcypb.System_PROCESS {
		step.CmpdOffset = cmpdOffset

		if !rkcy.IsKeylessStep(step) && !rkcy.IsInstanceStoreStep(step) {
			inst = plat.InstanceStore().GetInstance(step.Key)
		}

		if inst == nil && !rkcy.IsKeylessStep(step) && !rkcy.IsInstanceStoreStep(step) {
			// We should attempt to get the value from the DB, and we
			// do this by inserting a Storage READ step before this
			// one and sending things through again
			log.Debug().
				Str("TxnId", rtxn.Txn.Id).
				Str("Key", step.Key).
				Msg("Instance not found, injecting STORAGE READ")

			var err error
			if step.System == rkcypb.System_PROCESS && step.Command == rkcy.READ {
				// If we are doing a PROCESS READ, we can just substitute a STORAGE READ
				// The STORAGE READ, if successful, will inject a REFRESH_INSTANCE which
				// can safely stand in for the original PROCESS READ
				err = rtxn.ReplaceStep(
					rtxn.Txn.CurrentStepIdx,
					&rkcypb.ApecsTxn_Step{
						System:     rkcypb.System_STORAGE,
						Concern:    step.Concern,
						Command:    rkcy.READ,
						Key:        step.Key,
						CmpdOffset: cmpdOffset, // provide our PROCESS offset to the STORAGE step so it is recorded in the DB
					},
				)
			} else {
				// If not a PROCESS READ, we insert the STORAGE READ but preseve the original
				// command to run after the STORAGE READ runs
				err = rtxn.InsertSteps(
					rtxn.Txn.CurrentStepIdx,
					&rkcypb.ApecsTxn_Step{
						System:     rkcypb.System_STORAGE,
						Concern:    step.Concern,
						Command:    rkcy.READ,
						Key:        step.Key,
						CmpdOffset: cmpdOffset, // provide our PROCESS offset to the STORAGE step so it is recorded in the DB
					},
				)
			}
			if err != nil {
				log.Error().Err(err).Msg("error in InsertSteps")
				produceApecsTxnError(ctx, aprod, span, rtxn, nil, rkcypb.Code_INTERNAL, true, wg, "advanceApecsTxn Key=%s: Unable to insert Storage READ implicit steps", step.Key)
				return rkcypb.Code_INTERNAL
			}
			err = producer.ProduceCurrentStep(aprod, rtxn.Txn, rtxn.TraceParent, wg)
			if err != nil {
				produceApecsTxnError(ctx, aprod, span, rtxn, nil, rkcypb.Code_INTERNAL, true, wg, "advanceApecsTxn error=\"%s\": Failed produceCurrentStep for next step", err.Error())
				return rkcypb.Code_INTERNAL
			}
			return rkcypb.Code_OK
		} else if inst != nil && step.Command == rkcy.CREATE {
			produceApecsTxnError(ctx, aprod, span, rtxn, step, rkcypb.Code_INTERNAL, true, wg, "advanceApecsTxn Key=%s: Instance already exists in store in CREATE command", step.Key)
			return rkcypb.Code_INTERNAL
		}
	}

	if step.CmpdOffset == nil {
		produceApecsTxnError(ctx, aprod, span, rtxn, step, rkcypb.Code_INTERNAL, true, wg, "advanceApecsTxn: Nil offset")
		return rkcypb.Code_INTERNAL
	}

	args := &rkcy.StepArgs{
		TxnId:         rtxn.Txn.Id,
		Key:           step.Key,
		Instance:      inst,
		Payload:       step.Payload,
		EffectiveTime: step.EffectiveTime.AsTime(),
		CmpdOffset:    step.CmpdOffset,
	}

	var addSteps []*rkcypb.ApecsTxn_Step
	step.Result, addSteps = handleCommand(
		ctx,
		plat,
		step.Concern,
		step.System,
		step.Command,
		rtxn.Txn.Direction,
		args,
		confRdr,
		wg,
	)

	if (step.Result == nil || step.Result.Code != rkcypb.Code_OK) &&
		rtxn.Txn.UponError == rkcypb.UponError_BAILOUT {
		// We'll bailout up the chain, but down here we can at least
		// log any info we have about the result.
		var code rkcypb.Code
		if step.Result != nil {
			code = step.Result.Code
			for _, logEvt := range step.Result.LogEvents {
				log.Error().
					Str("TxnId", args.TxnId).
					Str("Concern", step.Concern).
					Str("System", step.System.String()).
					Str("Command", step.Command).
					Str("Direction", rtxn.Txn.Direction.String()).
					Msgf("BAILOUT LogEvent: %s %s", rkcypb.Severity_name[int32(logEvt.Sev)], logEvt.Msg)
			}
		} else {
			code = rkcypb.Code_NIL_RESULT
		}
		produceApecsTxnError(ctx, aprod, span, rtxn, step, rkcypb.Code_INTERNAL, true, wg, "advanceApecsTxn Key=%s: BAILOUT, %s", step.Key, jsonNoErr(step))
		return code
	}

	if step.Result == nil {
		produceApecsTxnError(ctx, aprod, span, rtxn, step, rkcypb.Code_INTERNAL, true, wg, "advanceApecsTxn Key=%s: nil result from step handler, %s", step.Key, jsonNoErr(step))
		return step.Result.Code
	}

	step.Result.ProcessedTime = now

	if step.Result.LogEvents != nil {
		for _, logEvt := range step.Result.LogEvents {
			sevMsg := fmt.Sprintf("%s %s", rkcypb.Severity_name[int32(logEvt.Sev)], logEvt.Msg)
			if logEvt.Sev == rkcypb.Severity_ERR {
				span.RecordError(errors.New(sevMsg))
			} else {
				span.AddEvent(sevMsg)
			}
		}
	}
	span.SetAttributes(
		attribute.String("rkcy.code", strconv.Itoa(int(step.Result.Code))),
	)
	if step.Result.Code != rkcypb.Code_OK {
		produceApecsTxnError(ctx, aprod, span, rtxn, step, step.Result.Code, false, wg, "advanceApecsTxn Key=%s: Step failed with non OK result, %s", step.Key, jsonNoErr(step))
		return step.Result.Code
	}

	// Unless explcitly set by the handler, always assume we should
	// pass step payload in result to next step
	if step.Result.Payload == nil {
		step.Result.Payload = step.Payload
	}

	// Record orignal instance in step if result has changed the instance
	if tp.System == rkcypb.System_PROCESS && step.Result.Instance != nil {
		step.Instance = inst
	}

	// if we have additional steps from handleCommand we should place
	// them in the transaction
	if addSteps != nil && len(addSteps) > 0 {
		err := rtxn.InsertSteps(rtxn.Txn.CurrentStepIdx+1, addSteps...)
		if err != nil {
			produceApecsTxnError(ctx, aprod, span, rtxn, step, rkcypb.Code_INTERNAL, true, wg, "InsertSteps failure: %s", err.Error())
			return rkcypb.Code_INTERNAL
		}
	}

	produceNextStep(ctx, plat, aprod, span, rtxn, step, cmpdOffset, wg)
	return step.Result.Code
}

func consumeApecsTopic(
	ctx context.Context,
	plat rkcy.Platform,
	consumerBrokers string,
	fullTopic string,
	partition int32,
	tp *rkcy.TopicParts,
	bailoutCh chan<- bool,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	platCh := make(chan *rkcy.PlatformMessage)
	consumer.ConsumePlatformTopic(
		ctx,
		plat,
		platCh,
		nil,
		wg,
	)
	platMsg := <-platCh
	plat.UpdateStorageTargets(platMsg)

	cncAdminReadyCh := make(chan bool)
	cncAdminCh := make(chan *consumer.ConcernAdminMessage)
	consumer.ConsumeConcernAdminTopic(
		ctx,
		plat,
		cncAdminCh,
		tp.Concern,
		cncAdminReadyCh,
		wg,
	)

	confMgr := config.NewConfigMgr(ctx, plat, wg)
	confRdr := config.NewConfigMgrRdr(confMgr)

	aprod := plat.NewApecsProducer(ctx, nil, wg)

	var groupName string
	if plat.System() == rkcypb.System_STORAGE {
		groupName = fmt.Sprintf("rkcy_apecs_%s_%s", fullTopic, plat.StorageTarget())
	} else {
		groupName = fmt.Sprintf("rkcy_apecs_%s", fullTopic)
	}

	kafkaLogCh := make(chan kafka.LogEvent)
	go rkcy.PrintKafkaLogs(ctx, kafkaLogCh)
	cons, err := plat.NewConsumer(consumerBrokers, groupName, kafkaLogCh)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("failed to plat.NewConsumer")
	}
	shouldCommit := false
	defer func() {
		log.Warn().
			Str("Topic", fullTopic).
			Msgf("CONSUMER Closing...")
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
			Msgf("CONSUMER CLOSED")
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

	txnsToCancel := make(map[string]time.Time)

	//	cncAdminReady := false
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
		case platMsg := <-platCh:
			plat.UpdateStorageTargets(platMsg)
		case <-cncAdminReadyCh:
			//			cncAdminReady = true
		case cncAdminMsg := <-cncAdminCh:
			if cncAdminMsg.Directive == rkcypb.Directive_CONCERN_ADMIN_CANCEL_TXN {
				log.Info().Msgf("CANCEL TXN %s", cncAdminMsg.ConcernAdminDirective.TxnId)
				txnsToCancel[cncAdminMsg.ConcernAdminDirective.TxnId] = time.Now()
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
					log.Debug().
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
					log.Debug().
						Str("Topic", fullTopic).
						Int32("Partition", partition).
						Int64("Offset", int64(msg.TopicPartition.Offset)).
						Msgf("Initial offset committed: %+v", comOffs)
				}

				directive := rkcy.GetDirective(msg)
				if directive == rkcypb.Directive_APECS_TXN {
					txn := rkcypb.ApecsTxn{}
					err := proto.Unmarshal(msg.Value, &txn)
					if err != nil {
						log.Error().
							Err(err).
							Msg("Failed to Unmarshal ApecsTxn")
					} else {
						offset := &rkcypb.CompoundOffset{
							Generation: tp.Generation,
							Partition:  partition,
							Offset:     int64(msg.TopicPartition.Offset),
						}

						rtxn, err := rkcy.NewRtApecsTxn(&txn, rkcy.GetTraceParent(msg))
						if err != nil {
							log.Error().
								Err(err).
								Msg("Failed to create RtApecsTxnf")
						} else {
							_, shouldCancel := txnsToCancel[rtxn.Txn.Id]
							if shouldCancel {
								delete(txnsToCancel, rtxn.Txn.Id)
								cancelApecsTxn(ctx, plat, rtxn, tp, offset, aprod, confRdr, wg)
							} else {
								code := advanceApecsTxn(ctx, plat, rtxn, tp, offset, aprod, confRdr, wg)
								// Bailout is necessary when a storage step fails.
								// Those must be retried indefinitely until success.
								// So... we force commit the offset to current and
								// kill ourselves, and we'll pick up where we left
								// off when we are restarted.
								if code != rkcypb.Code_OK && rtxn.Txn.UponError == rkcypb.UponError_BAILOUT {
									log.Error().
										Int("Code", int(code)).
										Str("TxnId", rtxn.Txn.Id).
										Str("Txn", jsonNoErr(proto.Message(rtxn.Txn))).
										Str("Topic", fullTopic).
										Int32("Partition", partition).
										Int64("Offset", int64(msg.TopicPartition.Offset)).
										Msgf("BAILOUT")

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

									bailoutCh <- true
									return
								}
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
	plat rkcy.Platform,
	adminBrokers string,
	consumerBrokers string,
	fullTopic string,
	partition int32,
	bailoutCh chan<- bool,
	wg *sync.WaitGroup,
) {
	tp, err := rkcy.ParseFullTopicName(fullTopic)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("Topic", fullTopic).
			Msg("startApecsRunner: failed to ParseFullTopicName")
	}

	if tp.System == rkcypb.System_NO_SYSTEM {
		log.Fatal().
			Str("Topic", fullTopic).
			Msg("startApecsRunner: No system matches topic")
	}

	if plat.System() != tp.System {
		log.Fatal().
			Str("Topic", fullTopic).
			Msgf("Mismatched system with topic: %s != %s", plat.System().String(), tp.System.String())
	}

	if tp.ConcernType != rkcypb.Concern_APECS {
		log.Fatal().
			Err(err).
			Str("Topic", fullTopic).
			Msg("startApecsRunner: not an APECS topic")
	}

	wg.Add(1)
	go consumeApecsTopic(
		ctx,
		plat,
		consumerBrokers,
		fullTopic,
		partition,
		tp,
		bailoutCh,
		wg,
	)
}

func handleCommand(
	ctx context.Context,
	plat rkcy.Platform,
	concern string,
	system rkcypb.System,
	command string,
	direction rkcypb.Direction,
	args *rkcy.StepArgs,
	confRdr rkcy.ConfigRdr,
	wg *sync.WaitGroup,
) (*rkcypb.ApecsTxn_Step_Result, []*rkcypb.ApecsTxn_Step) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Str("TxnId", args.TxnId).
				Str("Concern", concern).
				Str("System", system.String()).
				Str("Command", command).
				Str("Direction", direction.String()).
				Msgf("panic during handleCommand, %s, args: %+v", r, args)
			debug.PrintStack()
		}
	}()

	cncHdlr, ok := plat.ConcernHandlers()[concern]
	if !ok {
		rslt := &rkcypb.ApecsTxn_Step_Result{}
		rkcy.SetStepResult(rslt, fmt.Errorf("No handler for concern: '%s'", concern))
		return rslt, nil
	}

	switch system {
	case rkcypb.System_PROCESS:
		return cncHdlr.HandleLogicCommand(
			ctx,
			system,
			command,
			direction,
			args,
			plat.InstanceStore(),
			confRdr,
		)
	case rkcypb.System_STORAGE:
		fallthrough
	case rkcypb.System_STORAGE_SCND:
		return cncHdlr.HandleCrudCommand(
			ctx,
			system,
			command,
			direction,
			args,
			plat.StorageTarget(),
			wg,
		)
	default:
		rslt := &rkcypb.ApecsTxn_Step_Result{}
		rkcy.SetStepResult(rslt, fmt.Errorf("Invalid system: %s", system.String()))
		return rslt, nil
	}
}
