// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	otel_codes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/jsonutils"
	"github.com/lachlanorr/rocketcycle/version"
)

var (
	gInstanceStore *InstanceStore = NewInstanceStore()
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
		Str("TxnId", rtxn.txn.Id).
		Msg(logMsg)

	span.SetStatus(otel_codes.Error, logMsg)
	if logToResult {
		span.RecordError(errors.New(logMsg))
	}

	err := aprod.produceError(ctx, rtxn, step, code, logToResult, logMsg, wg)
	if err != nil {
		log.Error().
			Err(err).
			Str("TxnId", rtxn.txn.Id).
			Msg("produceApecsTxnError: Failed to produce error message")
	}
}

func produceNextStep(
	ctx context.Context,
	span trace.Span,
	rtxn *rtApecsTxn,
	step *ApecsTxn_Step,
	cmpdOffset *CompoundOffset,
	aprod *ApecsProducer,
	wg *sync.WaitGroup,
) {
	if rtxn.advanceStepIdx() {
		nextStep := rtxn.currentStep()
		// payload from last step should be passed to next step
		if nextStep.Payload == nil && IsPlatformCommand(nextStep.Command) {
			nextStep.Payload = step.Result.Payload
		}
		if nextStep.System == System_STORAGE && nextStep.CmpdOffset == nil {
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
		err := aprod.produceCurrentStep(rtxn.txn, rtxn.traceParent, wg)
		if err != nil {
			produceApecsTxnError(ctx, span, rtxn, nextStep, aprod, Code_INTERNAL, true, wg, "produceNextStep error=\"%s\": Failed produceCurrentStep for next step", err.Error())
			return
		}
	} else {
		// search for instance updates and create new storage txn to update storage
		if rtxn.txn.Direction == Direction_FORWARD {
			if gSettings.System == System_PROCESS {
				for _, step := range rtxn.txn.ForwardSteps {
					if step.System == System_STORAGE {
						// generate secondary storage steps
						if step.Command == CREATE || step.Command == UPDATE || step.Command == DELETE {
							storageStep := &ApecsTxn_Step{
								System:     System_STORAGE_SCND,
								Concern:    step.Concern,
								Command:    step.Command,
								Key:        step.Result.Key,
								Payload:    step.Result.Payload,
								CmpdOffset: step.CmpdOffset,
							}
							storageSteps := []*ApecsTxn_Step{storageStep}
							storageTxnId := NewTraceId()
							rtxn.txn.AssocTxns = append(rtxn.txn.AssocTxns, &AssocTxn{Id: storageTxnId, Type: AssocTxn_SECONDARY_STORAGE})

							err := aprod.executeTxn(
								storageTxnId,
								&AssocTxn{Id: rtxn.txn.Id, Type: AssocTxn_PARENT},
								rtxn.traceParent,
								UponError_BAILOUT,
								storageSteps,
								wg,
							)
							if err != nil {
								produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "produceNextStep error=\"%s\" Key=%s: Failed to apply secondary storage steps", err.Error(), step.Key)
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
			storageStepsMap := make(map[string]*ApecsTxn_Step)
			for _, step := range rtxn.txn.ForwardSteps {
				if step.System == System_PROCESS &&
					(step.Result.Instance != nil || step.Command == REQUEST_RELATED || step.Command == REFRESH_RELATED) {
					var stepInst []byte
					stepKey := fmt.Sprintf("%s_%s", step.Concern, step.Key)
					if step.Result.Instance != nil {
						stepInst = PackPayloads(step.Result.Instance, gInstanceStore.GetRelated(step.Key))
					} else if step.Command == REQUEST_RELATED {
						relRsp := &RelatedResponse{}
						err := proto.Unmarshal(step.Result.Payload, relRsp)
						if err != nil {
							log.Error().
								Err(err).
								Msgf("UPDATE_ASYNC ERROR: failed to decode RelatedResponse")
						} else {
							stepInst = relRsp.Payload
						}
					} else if step.Command == REFRESH_RELATED {
						stepInst = step.Result.Payload
					}
					if stepInst == nil {
						log.Error().
							Msgf("UPDATE_ASYNC ERROR: Empty stepInst, stepKey=%s", stepKey)
					} else {
						storageStepsMap[stepKey] = &ApecsTxn_Step{
							System:        System_STORAGE,
							Concern:       step.Concern,
							Command:       UPDATE_ASYNC,
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
				storageSteps := []*ApecsTxn_Step{step}
				storageTxnId := NewTraceId()
				rtxn.txn.AssocTxns = append(rtxn.txn.AssocTxns, &AssocTxn{Id: storageTxnId, Type: AssocTxn_GENERATED_STORAGE})

				err := aprod.executeTxn(
					storageTxnId,
					&AssocTxn{Id: rtxn.txn.Id, Type: AssocTxn_PARENT},
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
					// off failing very loudly.
					log.Fatal().
						Msgf("produceNextStep error=\"%s\" Key=%s: Failed to update storage", err.Error(), step.Key)

				} else {
					// also execute secondary storage txn
					storageSteps[0].System = System_STORAGE_SCND
					storageScndTxnId := NewTraceId()
					rtxn.txn.AssocTxns = append(rtxn.txn.AssocTxns, &AssocTxn{Id: storageTxnId, Type: AssocTxn_GENERATED_STORAGE})

					err := aprod.executeTxn(
						storageScndTxnId,
						&AssocTxn{Id: rtxn.txn.Id, Type: AssocTxn_PARENT},
						rtxn.traceParent,
						UponError_BAILOUT,
						storageSteps,
						wg,
					)
					if err != nil {
						produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "produceNextStep error=\"%s\" Key=%s: Failed to update secondary storage", err.Error(), step.Key)
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

		err := aprod.produceComplete(ctx, rtxn, wg)
		if err != nil {
			produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "produceNextStep error=\"%s\": Failed to produceComplete", err.Error())
			return
		}
	}
}

func isKeylessStep(step *ApecsTxn_Step) bool {
	return (step.System == System_STORAGE && step.Command == CREATE) ||
		(step.System == System_PROCESS && step.Command == VALIDATE_CREATE)
}

func isInstanceStoreStep(step *ApecsTxn_Step) bool {
	return step.System == System_PROCESS &&
		(step.Command == REFRESH_INSTANCE || step.Command == FLUSH_INSTANCE)
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
	rtxn *rtApecsTxn,
	tp *TopicParts,
	cmpdOffset *CompoundOffset,
	aprod *ApecsProducer,
	confRdr *ConfigRdr,
	wg *sync.WaitGroup,
) {
	ctx = InjectTraceParent(ctx, rtxn.traceParent)
	ctx, span, step := Telem().StartStep(ctx, rtxn)
	defer span.End()

	produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_CANCELLED, true, wg, "Txn Cancelled")
}

func advanceApecsTxn(
	ctx context.Context,
	rtxn *rtApecsTxn,
	tp *TopicParts,
	cmpdOffset *CompoundOffset,
	aprod *ApecsProducer,
	confRdr *ConfigRdr,
	wg *sync.WaitGroup,
) Code {
	ctx = InjectTraceParent(ctx, rtxn.traceParent)
	ctx, span, step := Telem().StartStep(ctx, rtxn)
	defer span.End()

	log.Debug().
		Str("TxnId", rtxn.txn.Id).
		Msgf("Advancing ApecsTxn: %s %d %s", Direction_name[int32(rtxn.txn.Direction)], rtxn.txn.CurrentStepIdx, jsonNoErr(step))

	if step.Result != nil {
		produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn Result=%+v: Current step already has Result, %s", step.Result, jsonNoErr(step))
		return Code_INTERNAL
	}

	if step.Concern != tp.Concern {
		produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn: Mismatched concern, expected=%s actual=%s, %s", tp.Concern, step.Concern, jsonNoErr(step))
		return Code_INTERNAL
	}

	if step.System != tp.System {
		produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn: Mismatched system, expected=%d actual=%d, %s", tp.System, step.System, jsonNoErr(step))
		return Code_INTERNAL
	}

	// Special case, only storage CREATE and process VALIDATE_CREATE can have empty key
	if step.Key == "" && !isKeylessStep(step) {
		prevStep := rtxn.previousStep()
		if prevStep != nil && prevStep.Result != nil && prevStep.Result.Key != "" {
			step.Key = prevStep.Result.Key
		} else {
			produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn: No key in step: %s %s, %s", step.System.String(), step.Command, jsonNoErr(step))
			return Code_INTERNAL
		}
	}

	// Grab current timestamp, which will be used in a couple places below
	now := timestamppb.Now()

	// Read instance from InstanceStore
	var inst []byte
	if step.System == System_PROCESS {
		step.CmpdOffset = cmpdOffset

		if !isKeylessStep(step) && !isInstanceStoreStep(step) {
			inst = gInstanceStore.GetInstance(step.Key)
		}

		if inst == nil && !isKeylessStep(step) && !isInstanceStoreStep(step) {
			// We should attempt to get the value from the DB, and we
			// do this by inserting a Storage READ step before this
			// one and sending things through again
			log.Debug().
				Str("TxnId", rtxn.txn.Id).
				Str("Key", step.Key).
				Msg("Instance not found, injecting STORAGE READ")

			var err error
			if step.System == System_PROCESS && step.Command == READ {
				// If we are doing a PROCESS READ, we can just substitute a STORAGE READ
				// The STORAGE READ, if successful, will inject a REFRESH_INSTANCE which
				// can safely stand in for the original PROCESS READ
				err = rtxn.replaceStep(
					rtxn.txn.CurrentStepIdx,
					&ApecsTxn_Step{
						System:     System_STORAGE,
						Concern:    step.Concern,
						Command:    READ,
						Key:        step.Key,
						CmpdOffset: cmpdOffset, // provide our PROCESS offset to the STORAGE step so it is recorded in the DB
					},
				)
			} else {
				// If not a PROCESS READ, we insert the STORAGE READ but preseve the original
				// command to run after the STORAGE READ runs
				err = rtxn.insertSteps(
					rtxn.txn.CurrentStepIdx,
					&ApecsTxn_Step{
						System:     System_STORAGE,
						Concern:    step.Concern,
						Command:    READ,
						Key:        step.Key,
						CmpdOffset: cmpdOffset, // provide our PROCESS offset to the STORAGE step so it is recorded in the DB
					},
				)
			}
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
			produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn Key=%s: Instance already exists in store in CREATE command", step.Key)
			return Code_INTERNAL
		}
	}

	if step.CmpdOffset == nil {
		produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn: Nil offset")
		return Code_INTERNAL
	}

	args := &StepArgs{
		TxnId:         rtxn.txn.Id,
		Key:           step.Key,
		Instance:      inst,
		Payload:       step.Payload,
		EffectiveTime: step.EffectiveTime.AsTime(),
		CmpdOffset:    step.CmpdOffset,
	}

	var addSteps []*ApecsTxn_Step
	step.Result, addSteps = handleCommand(
		ctx,
		step.Concern,
		step.System,
		step.Command,
		rtxn.txn.Direction,
		args,
		confRdr,
		wg,
	)

	if (step.Result == nil || step.Result.Code != Code_OK) &&
		rtxn.txn.UponError == UponError_BAILOUT {
		// We'll bailout up the chain, but down here we can at least
		// log any info we have about the result.
		var code Code
		if step.Result != nil {
			code = step.Result.Code
			for _, logEvt := range step.Result.LogEvents {
				log.Error().
					Str("TxnId", args.TxnId).
					Str("Concern", step.Concern).
					Str("System", step.System.String()).
					Str("Command", step.Command).
					Str("Direction", rtxn.txn.Direction.String()).
					Msgf("BAILOUT LogEvent: %s %s", Severity_name[int32(logEvt.Sev)], logEvt.Msg)
			}
		} else {
			code = Code_NIL_RESULT
		}
		produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn Key=%s: BAILOUT, %s", step.Key, jsonNoErr(step))
		return code
	}

	if step.Result == nil {
		produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "advanceApecsTxn Key=%s: nil result from step handler, %s", step.Key, jsonNoErr(step))
		return step.Result.Code
	}

	step.Result.ProcessedTime = now

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
		produceApecsTxnError(ctx, span, rtxn, step, aprod, step.Result.Code, false, wg, "advanceApecsTxn Key=%s: Step failed with non OK result, %s", step.Key, jsonNoErr(step))
		return step.Result.Code
	}

	// Unless explcitly set by the handler, always assume we should
	// pass step payload in result to next step
	if step.Result.Payload == nil {
		step.Result.Payload = step.Payload
	}

	// Record orignal instance in step if result has changed the instance
	if tp.System == System_PROCESS && step.Result.Instance != nil {
		step.Instance = inst
	}

	// if we have additional steps from handleCommand we should place
	// them in the transaction
	if addSteps != nil && len(addSteps) > 0 {
		err := rtxn.insertSteps(rtxn.txn.CurrentStepIdx+1, addSteps...)
		if err != nil {
			produceApecsTxnError(ctx, span, rtxn, step, aprod, Code_INTERNAL, true, wg, "insertSteps failure: %s", err.Error())
			return Code_INTERNAL
		}
	}

	produceNextStep(ctx, span, rtxn, step, cmpdOffset, aprod, wg)
	return step.Result.Code
}

func updateStorageTargets(platMsg *PlatformMessage) {
	if (platMsg.Directive & Directive_PLATFORM) != Directive_PLATFORM {
		log.Error().Msgf("Invalid directive for PlatformTopic: %s", platMsg.Directive.String())
		return
	}

	if IsStorageSystem(gSettings.System) {
		stgTgts := make(map[string]*StorageTarget)
		for _, platStgTgt := range platMsg.NewRtPlat.StorageTargets {
			tgt := &StorageTarget{
				Name:      platStgTgt.Name,
				Type:      platStgTgt.Type,
				IsPrimary: platStgTgt.IsPrimary,
				Config:    platStgTgt.Config,
			}
			if tgt.Config == nil {
				tgt.Config = make(map[string]string)
			}
			var ok bool
			tgt.Init, ok = gPlatformImpl.StorageInits[platStgTgt.Type]
			if !ok {
				log.Warn().
					Str("StorageTarget", platStgTgt.Name).
					Msgf("No StorageInit for target")
			}
			stgTgts[platStgTgt.Name] = tgt
		}
		for _, cncHdlr := range gConcernHandlers {
			cncHdlr.SetStorageTargets(stgTgts)
		}
	}
}

func consumeApecsTopic(
	ctx context.Context,
	adminBrokers string,
	consumerBrokers string,
	fullTopic string,
	partition int32,
	tp *TopicParts,
	bailoutCh chan<- bool,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	platCh := make(chan *PlatformMessage)
	consumePlatformTopic(
		ctx,
		platCh,
		gSettings.AdminBrokers,
		tp.Platform,
		tp.Environment,
		nil,
		wg,
	)
	platMsg := <-platCh
	updateStorageTargets(platMsg)

	cncAdminReadyCh := make(chan bool)
	cncAdminCh := make(chan *ConcernAdminMessage)
	consumeConcernAdminTopic(
		ctx,
		cncAdminCh,
		gSettings.AdminBrokers,
		tp.Platform,
		tp.Environment,
		tp.Concern,
		cncAdminReadyCh,
		wg,
	)

	confMgr := NewConfigMgr(ctx, adminBrokers, tp.Platform, tp.Environment, wg)
	confRdr := NewConfigRdr(confMgr)

	aprod := NewApecsProducer(ctx, adminBrokers, tp.Platform, tp.Environment, nil, wg)

	var groupName string
	if gSettings.System == System_STORAGE {
		groupName = fmt.Sprintf("rkcy_apecs_%s_%s", fullTopic, gSettings.StorageTarget)
	} else {
		groupName = fmt.Sprintf("rkcy_apecs_%s", fullTopic)
	}

	kafkaLogCh := make(chan kafka.LogEvent)
	go printKafkaLogs(ctx, kafkaLogCh)
	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        consumerBrokers,
		"group.id":                 groupName,
		"enable.auto.commit":       false,
		"enable.auto.offset.store": false,

		"go.logs.channel.enable": true,
		"go.logs.channel":        kafkaLogCh,
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
			updateStorageTargets(platMsg)
		case <-cncAdminReadyCh:
			//			cncAdminReady = true
		case cncAdminMsg := <-cncAdminCh:
			if cncAdminMsg.Directive == Directive_CONCERN_ADMIN_CANCEL_TXN {
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

				directive := GetDirective(msg)
				if directive == Directive_APECS_TXN {
					txn := ApecsTxn{}
					err := proto.Unmarshal(msg.Value, &txn)
					if err != nil {
						log.Error().
							Err(err).
							Msg("Failed to Unmarshal ApecsTxn")
					} else {
						offset := &CompoundOffset{
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
							_, shouldCancel := txnsToCancel[rtxn.txn.Id]
							if shouldCancel {
								delete(txnsToCancel, rtxn.txn.Id)
								cancelApecsTxn(ctx, rtxn, tp, offset, aprod, confRdr, wg)
							} else {
								code := advanceApecsTxn(ctx, rtxn, tp, offset, aprod, confRdr, wg)
								// Bailout is necessary when a storage step fails.
								// Those must be retried indefinitely until success.
								// So... we force commit the offset to current and
								// kill ourselves, and we'll pick up where we left
								// off when we are restarted.
								if code != Code_OK && rtxn.txn.UponError == UponError_BAILOUT {
									log.Error().
										Int("Code", int(code)).
										Str("TxnId", rtxn.txn.Id).
										Str("Txn", jsonNoErr(proto.Message(rtxn.txn))).
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
	adminBrokers string,
	consumerBrokers string,
	fullTopic string,
	partition int32,
	bailoutCh chan<- bool,
	wg *sync.WaitGroup,
) {
	tp, err := ParseFullTopicName(fullTopic)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("Topic", fullTopic).
			Msg("startApecsRunner: failed to ParseFullTopicName")
	}

	if tp.System == System_NO_SYSTEM {
		log.Fatal().
			Str("Topic", fullTopic).
			Msg("startApecsRunner: No system matches topic")
	}
	gSettings.System = tp.System

	if gSettings.System != tp.System {
		log.Fatal().
			Str("Topic", fullTopic).
			Msgf("Mismatched system with topic: %s != %s", gSettings.System.String(), tp.System.String())
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
		fullTopic,
		partition,
		tp,
		bailoutCh,
		wg,
	)
}

func cobraApecsConsumer(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Str("Topic", gSettings.Topic).
		Int32("Partition", gSettings.Partition).
		Msg("APECS Consumer Started")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	defer func() {
		signal.Stop(interruptCh)
		cancel()
	}()

	bailoutCh := make(chan bool)

	var wg sync.WaitGroup
	go startApecsRunner(
		ctx,
		gSettings.AdminBrokers,
		gSettings.ConsumerBrokers,
		gSettings.Topic,
		gSettings.Partition,
		bailoutCh,
		&wg,
	)

	select {
	case <-interruptCh:
		log.Warn().
			Str("System", gSettings.System.String()).
			Str("Topic", gSettings.Topic).
			Int32("Partition", gSettings.Partition).
			Msg("APECS Consumer Stopped, SIGINT")
		cancel()
		wg.Wait()
		return
	case <-bailoutCh:
		log.Warn().
			Str("System", gSettings.System.String()).
			Str("Topic", gSettings.Topic).
			Int32("Partition", gSettings.Partition).
			Msg("APECS Consumer Stopped, BAILOUT")
		cancel()
		wg.Wait()
		return
	}
}
