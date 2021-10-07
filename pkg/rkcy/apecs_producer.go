// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type Step struct {
	Concern string
	Command string
	Key     string
	Payload proto.Message
}

type ApecsProducer struct {
	ctx               context.Context
	adminBrokers      string
	platformName      string
	respTarget        *TopicTarget
	producers         map[string]map[StandardTopicName]*Producer
	producersMu       sync.Mutex
	respRegisterCh    chan *RespChan
	respConsumerClose context.CancelFunc
}

type RespChan struct {
	TraceId   string
	RespCh    chan *ApecsTxn
	StartTime time.Time
}

func NewApecsProducer(
	ctx context.Context,
	adminBrokers string,
	platformName string,
	respTarget *TopicTarget,
	wg *sync.WaitGroup,
) *ApecsProducer {
	aprod := &ApecsProducer{
		ctx:          ctx,
		adminBrokers: adminBrokers,
		platformName: platformName,
		respTarget:   respTarget,

		producers:      make(map[string]map[StandardTopicName]*Producer),
		respRegisterCh: make(chan *RespChan, 10),
	}

	if aprod.respTarget != nil {
		var respConsumerCtx context.Context
		respConsumerCtx, aprod.respConsumerClose = context.WithCancel(aprod.ctx)
		wg.Add(1)
		go aprod.consumeResponseTopic(respConsumerCtx, aprod.respTarget, wg)
	}

	return aprod
}

func (aprod *ApecsProducer) Close() {
	if aprod.respTarget != nil {
		aprod.respConsumerClose()
	}

	aprod.producersMu.Lock()
	defer aprod.producersMu.Unlock()
	for _, concernProds := range aprod.producers {
		for _, pdc := range concernProds {
			pdc.Close()
		}
	}

	aprod.producers = make(map[string]map[StandardTopicName]*Producer)
}

func (aprod *ApecsProducer) getProducer(
	concernName string,
	topicName StandardTopicName,
) (*Producer, error) {
	aprod.producersMu.Lock()
	defer aprod.producersMu.Unlock()

	concernProds, ok := aprod.producers[concernName]
	if !ok {
		concernProds = make(map[StandardTopicName]*Producer)
		aprod.producers[concernName] = concernProds
	}
	pdc, ok := concernProds[topicName]
	if !ok {
		pdc = NewProducer(aprod.ctx, aprod.adminBrokers, aprod.platformName, concernName, string(topicName))

		if pdc == nil {
			return nil, fmt.Errorf(
				"ApecsProducer.getProducer Brokers=%s Platform=%s Concern=%s Topic=%s: Failed to create Producer",
				aprod.adminBrokers,
				aprod.platformName,
				concernName,
				topicName,
			)
		}
		concernProds[topicName] = pdc
	}
	return pdc, nil
}

func (aprod *ApecsProducer) produceResponse(rtxn *rtApecsTxn) error {
	if rtxn.txn.ResponseTarget == nil {
		return nil
	}

	respTgt := rtxn.txn.ResponseTarget

	kMsg, err := kafkaMessage(&respTgt.Topic, respTgt.Partition, rtxn.txn, Directive_APECS_TXN, rtxn.traceParent)
	if err != nil {
		return err
	}

	prodCh := getProducerCh(respTgt.Brokers)
	prodCh <- kMsg

	return nil
}

func respondThroughChannel(traceId string, respCh chan *ApecsTxn, txn *ApecsTxn) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Str("TraceId", traceId).
				Msgf("recover while sending to respCh '%s'", r)
		}
	}()
	respCh <- txn
}

func (aprod *ApecsProducer) consumeResponseTopic(
	ctx context.Context,
	respTarget *TopicTarget,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	reqMap := make(map[string]*RespChan)

	groupName := fmt.Sprintf("rkcy_response_%s", respTarget.Topic)

	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        respTarget.Brokers,
		"group.id":                 groupName,
		"enable.auto.commit":       false, // we commit manually on an interval
		"enable.auto.offset.store": false, // we commit to local store after processeing to get at least once behavior
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Str("BoostrapServers", respTarget.Brokers).
			Str("GroupId", groupName).
			Msg("Unable to kafka.NewConsumer")
	}
	shouldCommit := false
	defer func() {
		log.Warn().
			Str("Topic", respTarget.Topic).
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
			Str("Topic", respTarget.Topic).
			Msgf("Closed kafka consumer")
	}()

	err = cons.Assign([]kafka.TopicPartition{
		{
			Topic:     &respTarget.Topic,
			Partition: respTarget.Partition,
			Offset:    kafka.OffsetStored,
		},
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to Assign")
	}

	firstMessage := true
	cleanupTicker := time.NewTicker(10 * time.Second)
	commitTicker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("consumeResponseTopic exiting, ctx.Done()")
			return
		case <-cleanupTicker.C:
			for traceId, respCh := range reqMap {
				now := time.Now()
				if now.Sub(respCh.StartTime) >= time.Second*60 {
					log.Warn().
						Str("TraceId", traceId).
						Msgf("Deleting request channel info, this is not normal and this transaction may have been lost")
					respondThroughChannel(traceId, respCh.RespCh, nil)
					delete(reqMap, traceId)
				}
			}
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

			// Read all registration messages here so we are sure to catch them before handling message
			moreRegistrations := true
			for moreRegistrations {
				select {
				case rch := <-aprod.respRegisterCh:
					_, ok := reqMap[rch.TraceId]
					if ok {
						log.Error().
							Str("TraceId", rch.TraceId).
							Msg("TraceId already registered for responses, replacing with new value")
					}
					reqMap[rch.TraceId] = rch
				default:
					moreRegistrations = false
				}
			}

			timedOut := err != nil && err.(kafka.Error).Code() == kafka.ErrTimedOut
			if err != nil && !timedOut {
				log.Error().
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
						Str("Topic", respTarget.Topic).
						Int32("Partition", respTarget.Partition).
						Int64("Offset", int64(msg.TopicPartition.Offset)).
						Msg("Initial commit to current offset")

					comOffs, err := cons.CommitOffsets([]kafka.TopicPartition{
						{
							Topic:     &respTarget.Topic,
							Partition: respTarget.Partition,
							Offset:    msg.TopicPartition.Offset,
						},
					})
					if err != nil {
						log.Fatal().
							Err(err).
							Str("Topic", respTarget.Topic).
							Int32("Partition", respTarget.Partition).
							Int64("Offset", int64(msg.TopicPartition.Offset)).
							Msgf("Unable to commit initial offset")
					}
					log.Info().
						Str("Topic", respTarget.Topic).
						Int32("Partition", respTarget.Partition).
						Int64("Offset", int64(msg.TopicPartition.Offset)).
						Msgf("Initial offset committed: %+v", comOffs)
				}

				directive := GetDirective(msg)
				if directive == Directive_APECS_TXN {
					traceId := GetTraceId(msg)
					respCh, ok := reqMap[traceId]
					if !ok {
						log.Error().
							Str("TraceId", traceId).
							Msg("TraceId not found in reqMap")
					} else {
						delete(reqMap, traceId)
						txn := ApecsTxn{}
						err := proto.Unmarshal(msg.Value, &txn)
						if err != nil {
							log.Error().
								Err(err).
								Str("TraceId", traceId).
								Msg("Failed to Unmarshal ApecsTxn")
						} else {
							respondThroughChannel(traceId, respCh.RespCh, &txn)
						}
					}
				} else {
					log.Warn().
						Int("Directive", int(directive)).
						Msg("Invalid directive on ApecsTxn topic")
				}

				_, err = cons.StoreOffsets([]kafka.TopicPartition{
					{
						Topic:     &respTarget.Topic,
						Partition: respTarget.Partition,
						Offset:    msg.TopicPartition.Offset + 1,
					},
				})
				if err != nil {
					log.Fatal().
						Err(err).
						Msgf("Unable to store offsets %s/%d/%d", respTarget.Topic, respTarget.Partition, msg.TopicPartition.Offset+1)
				}
				shouldCommit = true
			}
		}
	}
}

func waitForResponse(ctx context.Context, respCh <-chan *ApecsTxn, timeout time.Duration) (*ApecsTxn, error) {
	timer := time.NewTimer(timeout)

	select {
	case <-ctx.Done():
		return nil, errors.New("context closed")
	case <-timer.C:
		return nil, errors.New("time out waiting on response")
	case txn := <-respCh:
		return txn, nil
	}
}

func CodeTranslate(code Code) codes.Code {
	switch code {
	case Code_OK:
		return codes.OK
	case Code_NOT_FOUND:
		return codes.NotFound
	case Code_CONSTRAINT_VIOLATION:
		return codes.AlreadyExists

	case Code_INVALID_ARGUMENT:
		return codes.InvalidArgument

	case Code_INTERNAL:
		fallthrough
	case Code_MARSHAL_FAILED:
		fallthrough
	case Code_CONNECTION:
		fallthrough
	case Code_UNKNOWN_COMMAND:
		fallthrough
	default:
		return codes.Internal
	}
}

func (aprod *ApecsProducer) ExecuteTxnSync(
	ctx context.Context,
	canRevert bool,
	payload proto.Message,
	steps []Step,
	timeout time.Duration,
) (*ApecsTxn_Step_Result, error) {

	ctx, span := Telem().StartFunc(ctx)
	defer span.End()
	traceId := span.SpanContext().TraceID().String()

	if aprod.respConsumerClose == nil {
		return nil, errors.New("ExecuteTxnSync failure, no TopicTarget provided")
	}

	respCh := RespChan{
		TraceId:   traceId,
		RespCh:    make(chan *ApecsTxn),
		StartTime: time.Now(),
	}
	defer close(respCh.RespCh)
	aprod.respRegisterCh <- &respCh

	err := aprod.ExecuteTxnAsync(
		ctx,
		canRevert,
		payload,
		steps,
	)

	if err != nil {
		return nil, err
	}

	txnResp, err := waitForResponse(ctx, respCh.RespCh, timeout)
	if err != nil {
		return nil, err
	}
	if txnResp == nil {
		return nil, errors.New("nil txn received")
	}

	success, result := ApecsTxnResult(txnResp)
	if !success {
		details := make([]*anypb.Any, 0, 1)
		resultAny, err := anypb.New(result)
		if err != nil {
			span.RecordError(errors.New("Unable to convert result to Any"))
		} else {
			details = append(details, resultAny)
		}
		stat := spb.Status{
			Code:    int32(CodeTranslate(result.Code)),
			Message: "failure",
			Details: details,
		}
		errProto := status.ErrorProto(&stat)
		return nil, errProto
	}

	return result, nil
}

func (aprod *ApecsProducer) ExecuteTxnAsync(
	ctx context.Context,
	canRevert bool,
	payload proto.Message,
	steps []Step,
) error {
	stepsPb := make([]*ApecsTxn_Step, len(steps))
	for i, step := range steps {
		payloadBytes, err := proto.Marshal(step.Payload)
		if err != nil {
			return err
		}

		stepsPb[i] = &ApecsTxn_Step{
			System:  System_PROCESS,
			Concern: step.Concern,
			Command: step.Command,
			Key:     step.Key,
			Payload: payloadBytes,
		}
	}

	if payload != nil && len(stepsPb) > 0 {
		payloadBytes, err := proto.Marshal(payload)
		if err != nil {
			return err
		}
		stepsPb[0].Payload = payloadBytes
	}

	traceParent := ExtractTraceParent(ctx)
	if traceParent == "" {
		return fmt.Errorf("No SpanContext present in ctx")
	}
	traceId := TraceIdFromTraceParent(traceParent)

	return aprod.executeTxn(
		traceId,
		"",
		traceParent,
		canRevert,
		stepsPb,
	)
}

func (aprod *ApecsProducer) executeTxn(
	traceId string,
	assocTraceId string,
	traceParent string,
	canRevert bool,
	steps []*ApecsTxn_Step,
) error {
	txn, err := newApecsTxn(traceId, assocTraceId, aprod.respTarget, canRevert, steps)
	if err != nil {
		return err
	}

	return aprod.produceCurrentStep(txn, traceParent)
}

var gSystemToTopic = map[System]StandardTopicName{
	System_PROCESS: PROCESS,
	System_STORAGE: STORAGE,
}

func (aprod *ApecsProducer) produceError(rtxn *rtApecsTxn, step *ApecsTxn_Step, code Code, logToResult bool, msg string) error {
	if step == nil {
		step = rtxn.firstForwardStep()
		if step == nil {
			return fmt.Errorf("ApecsProducer.error TraceId=%s: failed to get firstForwardStep", rtxn.txn.TraceId)
		}
		msg = "DEFAULTING TO FIRST FORWARD STEP FOR LOGGING!!! - " + msg
	}

	// if no result, put one in so we can log to it
	if step.Result == nil {
		step.Result = &ApecsTxn_Step_Result{
			Code:          code,
			ProcessedTime: timestamppb.Now(),
			EffectiveTime: timestamppb.Now(),
		}
	}

	if logToResult {
		step.Result.LogEvents = append(
			step.Result.LogEvents,
			&LogEvent{
				Sev: Severity_ERR,
				Msg: msg,
			},
		)
	}

	prd, err := aprod.getProducer(step.Concern, ERROR)
	if err != nil {
		return err
	}

	txnSer, err := proto.Marshal(rtxn.txn)
	if err != nil {
		return err
	}

	prd.Produce(Directive_APECS_TXN, rtxn.traceParent, []byte(step.Key), txnSer, nil)

	err = aprod.produceResponse(rtxn)
	if err != nil {
		return err
	}

	return nil
}

func (aprod *ApecsProducer) produceComplete(rtxn *rtApecsTxn) error {
	// LORRNOTE 2021-03-21: Complete messages to to the concern of the
	// first step, which I think makes sense in most cases
	step := rtxn.firstForwardStep()
	if step == nil {
		return fmt.Errorf("ApecsProducer.complete TraceId=%s: failed to get firstForwardStep", rtxn.txn.TraceId)
	}

	prd, err := aprod.getProducer(step.Concern, COMPLETE)
	if err != nil {
		return err
	}

	txnSer, err := proto.Marshal(rtxn.txn)
	if err != nil {
		return err
	}

	prd.Produce(Directive_APECS_TXN, rtxn.traceParent, []byte(step.Key), txnSer, nil)

	err = aprod.produceResponse(rtxn)
	if err != nil {
		return err
	}

	return nil
}

func (aprod *ApecsProducer) produceCurrentStep(txn *ApecsTxn, traceParent string) error {
	rtxn, err := newRtApecsTxn(txn, traceParent)
	if err != nil {
		return err
	}

	step := rtxn.currentStep()

	var prd *Producer = nil
	topicName, ok := gSystemToTopic[step.System]
	if !ok {
		return fmt.Errorf("ApecsProducer.Process TraceId=%s System=%d: Invalid System", rtxn.txn.TraceId, step.System)
	}

	prd, err = aprod.getProducer(step.Concern, topicName)
	if err != nil {
		return err
	}

	txnSer, err := proto.Marshal(txn)
	if err != nil {
		return err
	}

	prd.Produce(Directive_APECS_TXN, traceParent, []byte(step.Key), txnSer, nil)
	return nil
}
