// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package apecs

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	otel_codes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/jsonutils"
	"github.com/lachlanorr/rocketcycle/pkg/mgmt"
	"github.com/lachlanorr/rocketcycle/pkg/platform"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/pkg/telem"
	"github.com/lachlanorr/rocketcycle/version"
)

func produceApecsTxnError(
	ctx context.Context,
	plat *platform.Platform,
	span trace.Span,
	rtxn *rkcy.RtApecsTxn,
	step *rkcypb.ApecsTxn_Step,
	code rkcypb.Code,
	logToResult bool,
	format string,
	args ...interface{},
) rkcypb.Code {
	logMsg := fmt.Sprintf(format, args...)

	log.Error().
		Str("TxnId", rtxn.Txn.Id).
		Msg(logMsg)

	span.SetStatus(otel_codes.Error, logMsg)
	if logToResult {
		span.RecordError(errors.New(logMsg))
	}

	err := ProduceError(
		ctx,
		plat,
		rtxn,
		step,
		code,
		logToResult,
		logMsg,
	)
	if err != nil {
		log.Error().
			Err(err).
			Str("TxnId", rtxn.Txn.Id).
			Msg("produceApecsTxnError: Failed to produce error message")
	}

	return code
}

func produceNextStep(
	ctx context.Context,
	plat *platform.Platform,
	span trace.Span,
	rtxn *rkcy.RtApecsTxn,
	step *rkcypb.ApecsTxn_Step,
	cmpdOffset *rkcypb.CompoundOffset,
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
		err := ProduceCurrentStep(ctx, plat.WaitGroup(), plat.ApecsProducer, rtxn.Txn, rtxn.TraceParent)
		if err != nil {
			produceApecsTxnError(
				ctx,
				plat,
				span,
				rtxn,
				nextStep,
				rkcypb.Code_INTERNAL,
				true,
				"produceNextStep error=\"%s\": Failed produceCurrentStep for next step",
				err.Error(),
			)
			return
		}
	} else { // txn is complete
		// search for instance updates and create new storage txn to update storage
		if rtxn.Txn.Direction == rkcypb.Direction_FORWARD {
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

						err := executeTxn(
							ctx,
							plat,
							storageTxnId,
							&rkcypb.AssocTxn{Id: rtxn.Txn.Id, Type: rkcypb.AssocTxn_PARENT},
							rtxn.TraceParent,
							rkcypb.UponError_BAILOUT,
							storageSteps,
							nil,
						)
						if err != nil {
							produceApecsTxnError(
								ctx,
								plat,
								span,
								rtxn,
								step,
								rkcypb.Code_INTERNAL,
								true,
								"produceNextStep error=\"%s\" Key=%s: Failed to apply secondary storage steps",
								err.Error(), step.Key,
							)
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

			// Generate update steps for any changed instances
			storageStepsMap := make(map[string]*rkcypb.ApecsTxn_Step)
			for _, step := range rtxn.Txn.ForwardSteps {
				if step.System == rkcypb.System_PROCESS &&
					(step.Result.Instance != nil || step.Command == rkcy.REQUEST_RELATED || step.Command == rkcy.REFRESH_RELATED) {
					var stepInst []byte
					stepKey := fmt.Sprintf("%s_%s", step.Concern, step.Key)
					if step.Result.Instance != nil {
						stepInst = rkcy.PackPayloads(step.Result.Instance, plat.InstanceStore.GetRelated(step.Key))
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

				err := executeTxn(
					ctx,
					plat,
					storageTxnId,
					&rkcypb.AssocTxn{Id: rtxn.Txn.Id, Type: rkcypb.AssocTxn_PARENT},
					rtxn.TraceParent,
					rkcypb.UponError_BAILOUT,
					storageSteps,
					nil,
				)
				if err != nil {
					produceApecsTxnError(
						ctx,
						plat,
						span,
						rtxn,
						step,
						rkcypb.Code_INTERNAL,
						true,
						"produceNextStep error=\"%s\" Key=%s: Failed to update storage",
						err.Error(),
						step.Key,
					)
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

					err := executeTxn(
						ctx,
						plat,
						storageScndTxnId,
						&rkcypb.AssocTxn{Id: rtxn.Txn.Id, Type: rkcypb.AssocTxn_PARENT},
						rtxn.TraceParent,
						rkcypb.UponError_BAILOUT,
						storageSteps,
						nil,
					)
					if err != nil {
						produceApecsTxnError(
							ctx,
							plat,
							span,
							rtxn,
							step,
							rkcypb.Code_INTERNAL,
							true,
							"produceNextStep error=\"%s\" Key=%s: Failed to update secondary storage",
							err.Error(),
							step.Key,
						)
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

		err := ProduceComplete(ctx, plat, rtxn)
		if err != nil {
			produceApecsTxnError(
				ctx,
				plat,
				span,
				rtxn,
				step,
				rkcypb.Code_INTERNAL,
				true,
				"produceNextStep error=\"%s\": Failed to ProduceComplete",
				err.Error(),
			)
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
	plat *platform.Platform,
	rtxn *rkcy.RtApecsTxn,
	tp *rkcy.TopicParts,
	cmpdOffset *rkcypb.CompoundOffset,
) {
	ctx = telem.InjectTraceParent(ctx, rtxn.TraceParent)
	ctx, span, step := telem.StartStep(ctx, rtxn)
	defer span.End()

	produceApecsTxnError(
		ctx,
		plat,
		span,
		rtxn,
		step,
		rkcypb.Code_CANCELLED,
		true,
		"Txn Cancelled",
	)
}

func isBailoutableCode(code rkcypb.Code) bool {
	return code != rkcypb.Code_OK &&
		code != rkcypb.Code_STEP_ALREADY_COMPLETE &&
		code != rkcypb.Code_MISMATCHED_CONCERN &&
		code != rkcypb.Code_MISMATCHED_SYSTEM &&
		code != rkcypb.Code_KEY_NOT_PRESENT
}

func advanceApecsTxn(
	ctx context.Context,
	plat *platform.Platform,
	storageTarget string,
	rtxn *rkcy.RtApecsTxn,
	tp *rkcy.TopicParts,
	cmpdOffset *rkcypb.CompoundOffset,
) rkcypb.Code {
	ctx = telem.InjectTraceParent(ctx, rtxn.TraceParent)
	ctx, span, step := telem.StartStep(ctx, rtxn)
	defer span.End()

	log.Debug().
		Str("TxnId", rtxn.Txn.Id).
		Msgf("Advancing ApecsTxn: %s %d %s", rkcypb.Direction_name[int32(rtxn.Txn.Direction)], rtxn.Txn.CurrentStepIdx, jsonNoErr(step))

	if step.Result != nil {
		return produceApecsTxnError(
			ctx,
			plat,
			span,
			rtxn,
			step,
			rkcypb.Code_STEP_ALREADY_COMPLETE,
			true,
			"advanceApecsTxn Result=%+v: Current step already has Result, %s",
			step.Result,
			jsonNoErr(step),
		)
	}

	if step.Concern != tp.Concern {
		return produceApecsTxnError(
			ctx,
			plat,
			span,
			rtxn,
			step,
			rkcypb.Code_MISMATCHED_CONCERN,
			true,
			"advanceApecsTxn: Mismatched concern, expected=%s actual=%s, %s",
			tp.Concern,
			step.Concern,
			jsonNoErr(step),
		)
	}

	if step.System != tp.System {
		return produceApecsTxnError(
			ctx,
			plat,
			span,
			rtxn,
			step,
			rkcypb.Code_MISMATCHED_SYSTEM,
			true,
			"advanceApecsTxn: Mismatched system, expected=%d actual=%d, %s",
			tp.System,
			step.System,
			jsonNoErr(step),
		)
	}

	// Special case, only storage CREATE and process VALIDATE_CREATE can have empty key
	if step.Key == "" && !rkcy.IsKeylessStep(step) {
		prevStep := rtxn.PreviousStep()
		if prevStep != nil && prevStep.Result != nil && prevStep.Result.Key != "" {
			step.Key = prevStep.Result.Key
		} else {
			return produceApecsTxnError(
				ctx,
				plat,
				span,
				rtxn,
				step,
				rkcypb.Code_KEY_NOT_PRESENT,
				true,
				"advanceApecsTxn: No key in step: %s %s, %s",
				step.System.String(),
				step.Command,
				jsonNoErr(step),
			)
		}
	}

	// Grab current timestamp, which will be used in a couple places below
	now := timestamppb.Now()

	// Read instance from InstanceStore
	var inst []byte
	if step.System == rkcypb.System_PROCESS {
		step.CmpdOffset = cmpdOffset

		if !rkcy.IsKeylessStep(step) && !rkcy.IsInstanceStoreStep(step) {
			inst = plat.InstanceStore.GetInstance(step.Key)
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
				return produceApecsTxnError(
					ctx,
					plat,
					span,
					rtxn,
					nil,
					rkcypb.Code_INTERNAL,
					true,
					"advanceApecsTxn Key=%s: Unable to insert Storage READ implicit steps",
					step.Key,
				)
			}
			err = ProduceCurrentStep(ctx, plat.WaitGroup(), plat.ApecsProducer, rtxn.Txn, rtxn.TraceParent)
			if err != nil {
				return produceApecsTxnError(
					ctx,
					plat,
					span,
					rtxn,
					nil,
					rkcypb.Code_INTERNAL,
					true,
					"advanceApecsTxn error=\"%s\": Failed produceCurrentStep for next step",
					err.Error(),
				)
			}
			return rkcypb.Code_OK
		} else if inst != nil && step.Command == rkcy.CREATE {
			return produceApecsTxnError(
				ctx,
				plat,
				span,
				rtxn,
				step,
				rkcypb.Code_INTERNAL,
				true,
				"advanceApecsTxn Key=%s: Instance already exists in store in CREATE command",
				step.Key,
			)
		}
	}

	if step.CmpdOffset == nil {
		return produceApecsTxnError(
			ctx,
			plat,
			span,
			rtxn,
			step,
			rkcypb.Code_INTERNAL,
			true,
			"advanceApecsTxn: Nil offset",
		)
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
		storageTarget,
		step.Concern,
		step.System,
		step.Command,
		rtxn.Txn.Direction,
		args,
	)

	if (step.Result == nil || isBailoutableCode(step.Result.Code)) &&
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
		return produceApecsTxnError(
			ctx,
			plat,
			span,
			rtxn,
			step,
			code,
			true,
			"advanceApecsTxn Key=%s: BAILOUT, %s",
			step.Key,
			jsonNoErr(step),
		)
	}

	if step.Result == nil {
		return produceApecsTxnError(
			ctx,
			plat,
			span,
			rtxn,
			step,
			rkcypb.Code_INTERNAL,
			true,
			"advanceApecsTxn Key=%s: nil result from step handler, %s",
			step.Key,
			jsonNoErr(step),
		)
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
		return produceApecsTxnError(
			ctx,
			plat,
			span,
			rtxn,
			step,
			step.Result.Code,
			false,
			"advanceApecsTxn Key=%s: Step failed with non OK result, %s",
			step.Key,
			jsonNoErr(step),
		)
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
			return produceApecsTxnError(
				ctx,
				plat,
				span,
				rtxn,
				step,
				rkcypb.Code_INTERNAL,
				true,
				"InsertSteps failure: %s",
				err.Error(),
			)
		}
	}

	produceNextStep(ctx, plat, span, rtxn, step, cmpdOffset)
	return step.Result.Code
}

func consumeApecsTopic(
	ctx context.Context,
	plat *platform.Platform,
	storageTarget string,
	consumerBrokers string,
	tp *rkcy.TopicParts,
	partition int32,
	bailoutCh chan<- bool,
) {
	defer plat.WaitGroup().Done()

	platCh := make(chan *mgmt.PlatformMessage)
	mgmt.ConsumePlatformTopic(
		ctx,
		plat.WaitGroup(),
		plat.StreamProvider(),
		plat.Args.Platform,
		plat.Args.Environment,
		plat.Args.AdminBrokers,
		platCh,
		nil,
	)
	platMsg := <-platCh
	if rkcy.IsStorageSystem(tp.System) {
		plat.UpdateStorageTargets(platMsg)
	}

	cncAdminReadyCh := make(chan bool)
	cncAdminCh := make(chan *mgmt.ConcernAdminMessage)
	mgmt.ConsumeConcernAdminTopic(
		ctx,
		plat.WaitGroup(),
		plat.StreamProvider(),
		plat.Args.Platform,
		plat.Args.Environment,
		plat.Args.AdminBrokers,
		cncAdminCh,
		tp.Concern,
		cncAdminReadyCh,
	)

	var groupName string
	if tp.System == rkcypb.System_STORAGE {
		groupName = fmt.Sprintf("rkcy_apecs_%s_%s", tp.FullTopic, storageTarget)
	} else {
		groupName = fmt.Sprintf("rkcy_apecs_%s", tp.FullTopic)
	}

	kafkaLogCh := make(chan kafka.LogEvent)
	go rkcy.PrintKafkaLogs(ctx, kafkaLogCh)
	cons, err := plat.StreamProvider().NewConsumer(consumerBrokers, groupName, kafkaLogCh)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("failed to plat.NewConsumer")
	}
	shouldCommit := false
	defer func() {
		log.Warn().
			Str("Topic", tp.FullTopic).
			Msgf("CONSUMER Closing...")

		// make sure we don't have pending values on cncAdminReadyCh
		select {
		case <-cncAdminReadyCh:
			log.Warn().
				Str("Topic", tp.FullTopic).
				Msgf("Pending cncAdminReadyCh during close")
		default:
		}

		if shouldCommit {
			_, err = cons.Commit()
			shouldCommit = false
			if err != nil {
				log.Error().
					Err(err).
					Msgf("Unable to commit")
			}
		}
		err := cons.Close()
		if err != nil {
			log.Error().
				Err(err).
				Str("Topic", tp.FullTopic).
				Msgf("Error during consumer.Close()")
		}
		log.Warn().
			Str("Topic", tp.FullTopic).
			Msgf("CONSUMER CLOSED")
	}()

	err = cons.Assign([]kafka.TopicPartition{
		{
			Topic:     &tp.FullTopic,
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
			if rkcy.IsStorageSystem(tp.System) {
				plat.UpdateStorageTargets(platMsg)
			}
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
						Str("Topic", tp.FullTopic).
						Int32("Partition", partition).
						Int64("Offset", int64(msg.TopicPartition.Offset)).
						Msg("Initial commit to current offset")

					comOffs, err := cons.CommitOffsets([]kafka.TopicPartition{
						{
							Topic:     &tp.FullTopic,
							Partition: partition,
							Offset:    msg.TopicPartition.Offset,
						},
					})
					if err != nil {
						log.Fatal().
							Err(err).
							Str("Topic", tp.FullTopic).
							Int32("Partition", partition).
							Int64("Offset", int64(msg.TopicPartition.Offset)).
							Msgf("Unable to commit initial offset")
					}
					log.Debug().
						Str("Topic", tp.FullTopic).
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
								cancelApecsTxn(ctx, plat, rtxn, tp, offset)
							} else {
								code := advanceApecsTxn(ctx, plat, storageTarget, rtxn, tp, offset)
								// Bailout is necessary when a storage step fails.
								// Those must be retried indefinitely until success.
								// So... we force commit the offset to current and
								// kill ourselves, and we'll pick up where we left
								// off when we are restarted.
								if rtxn.Txn.UponError == rkcypb.UponError_BAILOUT && isBailoutableCode(code) {
									log.Error().
										Int("Code", int(code)).
										Str("TxnId", rtxn.Txn.Id).
										Str("Txn", jsonNoErr(proto.Message(rtxn.Txn))).
										Str("Topic", tp.FullTopic).
										Int32("Partition", partition).
										Int64("Offset", int64(msg.TopicPartition.Offset)).
										Msgf("BAILOUT")

									_, err := cons.CommitOffsets([]kafka.TopicPartition{
										{
											Topic:     &tp.FullTopic,
											Partition: partition,
											Offset:    msg.TopicPartition.Offset,
										},
									})
									if err != nil {
										log.Error().
											Err(err).
											Str("Topic", tp.FullTopic).
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
						Topic:     &tp.FullTopic,
						Partition: partition,
						Offset:    msg.TopicPartition.Offset + 1,
					},
				})
				if err != nil {
					log.Fatal().
						Err(err).
						Msgf("Unable to store offsets %s/%d/%d", tp.FullTopic, partition, msg.TopicPartition.Offset+1)
				}
				shouldCommit = true
			}
		}
	}
}

func handleCommand(
	ctx context.Context,
	plat *platform.Platform,
	storageTarget string,
	concern string,
	system rkcypb.System,
	command string,
	direction rkcypb.Direction,
	args *rkcy.StepArgs,
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

	cncHdlr, ok := plat.ConcernHandlers[concern]
	if !ok {
		rslt := &rkcypb.ApecsTxn_Step_Result{}
		SetStepResult(rslt, fmt.Errorf("No handler for concern: '%s'", concern))
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
			plat.InstanceStore,
			plat.ConfigRdr(),
		)
	case rkcypb.System_STORAGE:
		fallthrough
	case rkcypb.System_STORAGE_SCND:
		return cncHdlr.HandleCrudCommand(
			ctx,
			plat.WaitGroup(),
			system,
			command,
			direction,
			args,
			storageTarget,
		)
	default:
		rslt := &rkcypb.ApecsTxn_Step_Result{}
		SetStepResult(rslt, fmt.Errorf("Invalid system: %s", system.String()))
		return rslt, nil
	}
}

func StartApecsRunner(
	ctx context.Context,
	plat *platform.Platform,
	storageTarget string,
	consumerBrokers string,
	tp *rkcy.TopicParts,
	partition int32,
	bailoutCh chan<- bool,
) {
	if tp.System == rkcypb.System_NO_SYSTEM {
		log.Fatal().
			Str("System", tp.System.String()).
			Str("Topic", tp.FullTopic).
			Msg("startApecsRunner: No system matches topic")
	}

	if tp.ConcernType != rkcypb.Concern_APECS {
		log.Fatal().
			Str("System", tp.System.String()).
			Str("Topic", tp.FullTopic).
			Msg("startApecsRunner: not an APECS topic")
	}

	plat.WaitGroup().Add(1)
	go consumeApecsTopic(
		ctx,
		plat,
		storageTarget,
		consumerBrokers,
		tp,
		partition,
		bailoutCh,
	)
}

func StartApecsConsumer(
	ctx context.Context,
	plat *platform.Platform,
	storageTarget string,
	consumerBrokers string,
	fullTopic string,
	partition int32,
	bailoutCh chan<- bool,
) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Str("Topic", fullTopic).
		Int32("Partition", partition).
		Msg("APECS Consumer Started")

	tp, err := rkcy.ParseFullTopicName(fullTopic)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("Topic", fullTopic).
			Msg("startApecsRunner: failed to ParseFullTopicName")
	}

	plat.InitConfigMgr(ctx)

	go StartApecsRunner(
		ctx,
		plat,
		storageTarget,
		consumerBrokers,
		tp,
		partition,
		bailoutCh,
	)

	select {
	case <-ctx.Done():
		return
	}
}
