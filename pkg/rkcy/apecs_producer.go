// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/consts"
)

type Step struct {
	Concern string
	Command string
	Key     string
	Payload proto.Message
}

type ApecsProducer struct {
	ctx               context.Context
	bootstrapServers  string
	platformName      string
	respTarget        *TopicTarget
	producers         map[string]map[consts.StandardTopicName]*Producer
	respProducers     map[string]*kafka.Producer
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
	bootstrapServers string,
	platformName string,
	respTarget *TopicTarget,
) *ApecsProducer {

	aprod := &ApecsProducer{
		ctx:              ctx,
		bootstrapServers: bootstrapServers,
		platformName:     platformName,
		respTarget:       respTarget,

		producers:      make(map[string]map[consts.StandardTopicName]*Producer),
		respProducers:  make(map[string]*kafka.Producer),
		respRegisterCh: make(chan *RespChan, 10),
	}

	if aprod.respTarget != nil {
		var respConsumerCtx context.Context
		respConsumerCtx, aprod.respConsumerClose = context.WithCancel(aprod.ctx)
		go aprod.consumeResponseTopic(respConsumerCtx, aprod.respTarget)
	}

	return aprod
}

func (aprod *ApecsProducer) Close() {
	if aprod.respTarget != nil {
		aprod.respConsumerClose()
	}

	for _, concernProds := range aprod.producers {
		for _, pdc := range concernProds {
			pdc.Close()
		}
	}

	for _, responseProd := range aprod.respProducers {
		responseProd.Close()
	}

	aprod.producers = make(map[string]map[consts.StandardTopicName]*Producer)
}

func (aprod *ApecsProducer) getProducer(
	concernName string,
	topicName consts.StandardTopicName,
) (*Producer, error) {
	concernProds, ok := aprod.producers[concernName]
	if !ok {
		concernProds = make(map[consts.StandardTopicName]*Producer)
		aprod.producers[concernName] = concernProds
	}
	pdc, ok := concernProds[topicName]
	if !ok {
		pdc = NewProducer(aprod.ctx, aprod.bootstrapServers, aprod.platformName, concernName, string(topicName))

		if pdc == nil {
			return nil, fmt.Errorf(
				"ApecsProducer.getProducer BootstrapServers=%s Platform=%s Concern=%s Topic=%s: Failed to create Producer",
				aprod.bootstrapServers,
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

	txnSer, err := proto.Marshal(rtxn.txn)
	if err != nil {
		return err
	}

	kProd, ok := aprod.respProducers[respTgt.BootstrapServers]
	if !ok {
		var err error
		kProd, err = kafka.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": respTgt.BootstrapServers,
		})
		if err != nil {
			return err
		}

		aprod.respProducers[respTgt.BootstrapServers] = kProd
	}

	kMsg := kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &respTgt.Topic,
			Partition: respTgt.Partition,
		},
		Value:   txnSer,
		Headers: standardHeaders(Directive_APECS_TXN, rtxn.traceParent),
	}

	err = kProd.Produce(&kMsg, nil)
	if err != nil {
		return err
	}

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

func (aprod *ApecsProducer) consumeResponseTopic(ctx context.Context, respTarget *TopicTarget) {
	reqMap := make(map[string]*RespChan)

	groupName := fmt.Sprintf("rkcy_%s_edge__%s_%d", aprod.platformName, respTarget.Topic, respTarget.Partition)

	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        respTarget.BootstrapServers,
		"group.id":                 groupName,
		"enable.auto.commit":       true, // librdkafka will commit to brokers for us on an interval and when we close consumer
		"enable.auto.offset.store": true, // librdkafka will commit to local store to get "at most once" behavior
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Str("BoostrapServers", respTarget.BootstrapServers).
			Str("GroupId", groupName).
			Msg("Unable to kafka.NewConsumer")
	}
	defer cons.Close()

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

	cleanupTicker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("watchTopic.consume exiting, ctx.Done()")
			return
		case <-cleanupTicker.C:
			for traceId, respCh := range reqMap {
				now := time.Now()
				if now.Sub(respCh.StartTime) >= time.Second*10 {
					log.Warn().
						Str("TraceId", traceId).
						Msgf("Deleting request channel info, this is not normal and this transaction may have been lost")
					respCh.RespCh <- nil
					delete(reqMap, traceId)
				}
			}
		default:
			msg, err := cons.ReadMessage(time.Millisecond * 10)

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

var systemToTopic = map[System]consts.StandardTopicName{
	System_PROCESS: consts.Process,
	System_STORAGE: consts.Storage,
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

	prd, err := aprod.getProducer(step.Concern, consts.Error)
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

	prd, err := aprod.getProducer(step.Concern, consts.Complete)
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
	topicName, ok := systemToTopic[step.System]
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
