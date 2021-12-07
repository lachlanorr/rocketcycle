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

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type Txn struct {
	Revert RevertType
	Steps  []Step
}

type Step struct {
	Concern       string
	Command       string
	Key           string
	Payload       proto.Message
	EffectiveTime time.Time
}

type ResultProto struct {
	Type     string
	Instance proto.Message
	Related  proto.Message
}

type RevertType int

const (
	Revertable    RevertType = 0
	NonRevertable            = 1
)

const defaultTimeout time.Duration = time.Minute * 5

func ValidateTxn(txn *Txn) error {
	if txn == nil {
		return fmt.Errorf("Nil txn")
	}
	if txn.Steps == nil || len(txn.Steps) == 0 {
		return fmt.Errorf("No steps in txn")
	}

	for _, s := range txn.Steps {
		if IsTxnProhibitedCommand(s.Command) {
			return fmt.Errorf("Invalid step command: %s", s.Command)
		}
	}

	return nil
}

type ApecsKafkaProducer struct {
	ctx               context.Context
	plat              Platform
	respTarget        *rkcypb.TopicTarget
	producers         map[string]map[StandardTopicName]Producer
	producersMtx      sync.Mutex
	respRegisterCh    chan *RespChan
	respConsumerClose context.CancelFunc
}

func (akprod *ApecsKafkaProducer) Platform() Platform {
	return akprod.plat
}

func (akprod *ApecsKafkaProducer) ResponseTarget() *rkcypb.TopicTarget {
	return akprod.respTarget
}

func (akprod *ApecsKafkaProducer) RegisterResponseChannel(respCh *RespChan) {
	akprod.respRegisterCh <- respCh
}

type RespChan struct {
	TxnId     string
	RespCh    chan *rkcypb.ApecsTxn
	StartTime time.Time
}

func NewApecsKafkaProducer(
	ctx context.Context,
	plat Platform,
	respTarget *rkcypb.TopicTarget,
	wg *sync.WaitGroup,
) *ApecsKafkaProducer {
	akprod := &ApecsKafkaProducer{
		ctx:        ctx,
		plat:       plat,
		respTarget: respTarget,

		producers:      make(map[string]map[StandardTopicName]Producer),
		respRegisterCh: make(chan *RespChan, 10),
	}

	if respTarget != nil {
		var respConsumerCtx context.Context
		respConsumerCtx, akprod.respConsumerClose = context.WithCancel(akprod.ctx)
		wg.Add(1)
		go akprod.consumeResponseTopic(respConsumerCtx, wg)
	}

	return akprod
}

func (akprod *ApecsKafkaProducer) Close() {
	if akprod.respTarget != nil {
		akprod.respConsumerClose()
	}

	akprod.producersMtx.Lock()
	defer akprod.producersMtx.Unlock()
	for _, concernProds := range akprod.producers {
		for _, pdc := range concernProds {
			pdc.Close()
		}
	}

	akprod.producers = make(map[string]map[StandardTopicName]Producer)
}

func (akprod *ApecsKafkaProducer) GetProducer(
	concernName string,
	topicName StandardTopicName,
	wg *sync.WaitGroup,
) (Producer, error) {
	akprod.producersMtx.Lock()
	defer akprod.producersMtx.Unlock()

	concernProds, ok := akprod.producers[concernName]
	if !ok {
		concernProds = make(map[StandardTopicName]Producer)
		akprod.producers[concernName] = concernProds
	}
	pdc, ok := concernProds[topicName]
	if !ok {
		pdc = akprod.plat.NewProducer(
			akprod.ctx,
			concernName,
			string(topicName),
			wg,
		)

		if pdc == nil {
			return nil, fmt.Errorf(
				"ApecsKafkaProducer.GetProducer Brokers=%s Platform=%s Concern=%s Topic=%s: Failed to create Producer",
				akprod.plat.AdminBrokers(),
				akprod.plat.Name(),
				concernName,
				topicName,
			)
		}
		concernProds[topicName] = pdc
	}
	return Producer(pdc), nil
}

func produceResponse(
	ctx context.Context,
	plat Platform,
	respBrokers string,
	rtxn *rtApecsTxn,
	wg *sync.WaitGroup,
) error {
	if rtxn.txn.ResponseTarget == nil {
		return nil
	}

	respTgt := rtxn.txn.ResponseTarget

	kMsg, err := newKafkaMessage(&respTgt.Topic, respTgt.Partition, rtxn.txn, rkcypb.Directive_APECS_TXN, rtxn.traceParent)
	if err != nil {
		return err
	}

	prodCh := plat.GetProducerCh(ctx, respBrokers, wg)
	prodCh <- kMsg

	return nil
}

func respondThroughChannel(txnId string, respCh chan *rkcypb.ApecsTxn, txn *rkcypb.ApecsTxn) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Str("TxnId", txnId).
				Msgf("recover while sending to respCh '%s'", r)
		}
	}()
	respCh <- txn
}

func (akprod *ApecsKafkaProducer) consumeResponseTopic(
	ctx context.Context,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	reqMap := make(map[string]*RespChan)

	groupName := fmt.Sprintf("rkcy_response_%s", akprod.respTarget.Topic)

	kafkaLogCh := make(chan kafka.LogEvent)
	go printKafkaLogs(ctx, kafkaLogCh)

	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        akprod.respTarget.Brokers,
		"group.id":                 groupName,
		"enable.auto.commit":       false,
		"enable.auto.offset.store": false,

		"go.logs.channel.enable": true,
		"go.logs.channel":        kafkaLogCh,
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Str("BoostrapServers", akprod.respTarget.Brokers).
			Str("GroupId", groupName).
			Msg("Unable to kafka.NewConsumer")
	}
	shouldCommit := false
	defer func() {
		log.Warn().
			Str("Topic", akprod.respTarget.Topic).
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
			Str("Topic", akprod.respTarget.Topic).
			Msgf("CONSUMER CLOSED")
	}()

	err = cons.Assign([]kafka.TopicPartition{
		{
			Topic:     &akprod.respTarget.Topic,
			Partition: akprod.respTarget.Partition,
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
			log.Warn().
				Msg("consumeResponseTopic exiting, ctx.Done()")
			return
		case <-cleanupTicker.C:
			for txnId, respCh := range reqMap {
				now := time.Now()
				if now.Sub(respCh.StartTime) >= time.Second*60 {
					log.Warn().
						Str("TxnId", txnId).
						Msgf("Deleting request channel info, this is not normal and this transaction may have been lost")
					respondThroughChannel(txnId, respCh.RespCh, nil)
					delete(reqMap, txnId)
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
				case rch := <-akprod.respRegisterCh:
					_, ok := reqMap[rch.TxnId]
					if ok {
						log.Error().
							Str("TxnId", rch.TxnId).
							Msg("TxnId already registered for responses, replacing with new value")
					}
					reqMap[rch.TxnId] = rch
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
					log.Debug().
						Str("Topic", akprod.respTarget.Topic).
						Int32("Partition", akprod.respTarget.Partition).
						Int64("Offset", int64(msg.TopicPartition.Offset)).
						Msg("Initial commit to current offset")

					comOffs, err := cons.CommitOffsets([]kafka.TopicPartition{
						{
							Topic:     &akprod.respTarget.Topic,
							Partition: akprod.respTarget.Partition,
							Offset:    msg.TopicPartition.Offset,
						},
					})
					if err != nil {
						log.Fatal().
							Err(err).
							Str("Topic", akprod.respTarget.Topic).
							Int32("Partition", akprod.respTarget.Partition).
							Int64("Offset", int64(msg.TopicPartition.Offset)).
							Msgf("Unable to commit initial offset")
					}
					log.Debug().
						Str("Topic", akprod.respTarget.Topic).
						Int32("Partition", akprod.respTarget.Partition).
						Int64("Offset", int64(msg.TopicPartition.Offset)).
						Msgf("Initial offset committed: %+v", comOffs)
				}

				directive := GetDirective(msg)
				if directive == rkcypb.Directive_APECS_TXN {
					txnId := GetTraceId(msg)
					respCh, ok := reqMap[txnId]
					if !ok {
						log.Error().
							Str("TxnId", txnId).
							Msg("TxnId not found in reqMap")
					} else {
						delete(reqMap, txnId)
						txn := rkcypb.ApecsTxn{}
						err := proto.Unmarshal(msg.Value, &txn)
						if err != nil {
							log.Error().
								Err(err).
								Str("TxnId", txnId).
								Msg("Failed to Unmarshal ApecsTxn")
						} else {
							respondThroughChannel(txnId, respCh.RespCh, &txn)
						}
					}
				} else {
					log.Warn().
						Int("Directive", int(directive)).
						Msg("Invalid directive on ApecsTxn topic")
				}

				_, err = cons.StoreOffsets([]kafka.TopicPartition{
					{
						Topic:     &akprod.respTarget.Topic,
						Partition: akprod.respTarget.Partition,
						Offset:    msg.TopicPartition.Offset + 1,
					},
				})
				if err != nil {
					log.Fatal().
						Err(err).
						Msgf("Unable to store offsets %s/%d/%d", akprod.respTarget.Topic, akprod.respTarget.Partition, msg.TopicPartition.Offset+1)
				}
				shouldCommit = true
			}
		}
	}
}

func waitForResponse(ctx context.Context, respCh <-chan *rkcypb.ApecsTxn, timeout time.Duration) (*rkcypb.ApecsTxn, error) {
	if timeout.Seconds() == 0 {
		timeout = defaultTimeout
	}
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

func CodeTranslate(code rkcypb.Code) codes.Code {
	switch code {
	case rkcypb.Code_OK:
		return codes.OK
	case rkcypb.Code_NOT_FOUND:
		return codes.NotFound
	case rkcypb.Code_CONSTRAINT_VIOLATION:
		return codes.AlreadyExists

	case rkcypb.Code_INVALID_ARGUMENT:
		return codes.InvalidArgument

	case rkcypb.Code_INTERNAL:
		fallthrough
	case rkcypb.Code_MARSHAL_FAILED:
		fallthrough
	case rkcypb.Code_CONNECTION:
		fallthrough
	case rkcypb.Code_UNKNOWN_COMMAND:
		fallthrough
	default:
		return codes.Internal
	}
}

type preparedApecsSteps struct {
	traceParent string
	txnId       string
	uponError   rkcypb.UponError
	steps       []*rkcypb.ApecsTxn_Step
}

func prepareApecsSteps(
	ctx context.Context,
	txn *Txn,
	telem *Telemetry,
	wg *sync.WaitGroup,
) (*preparedApecsSteps, error) {
	if err := ValidateTxn(txn); err != nil {
		return nil, err
	}

	prepSteps := &preparedApecsSteps{}

	prepSteps.traceParent = ExtractTraceParent(ctx)
	if prepSteps.traceParent == "" {
		return nil, fmt.Errorf("No SpanContext present in ctx")
	}
	prepSteps.txnId = TraceIdFromTraceParent(prepSteps.traceParent)

	if txn.Revert == Revertable {
		prepSteps.uponError = rkcypb.UponError_REVERT
	} else {
		prepSteps.uponError = rkcypb.UponError_REPORT
	}

	prepSteps.steps = make([]*rkcypb.ApecsTxn_Step, len(txn.Steps))
	for i, step := range txn.Steps {
		payloadBytes, err := proto.Marshal(step.Payload)
		if err != nil {
			return nil, err
		}

		if step.EffectiveTime.IsZero() {
			step.EffectiveTime = time.Now()
		}

		prepSteps.steps[i] = &rkcypb.ApecsTxn_Step{
			System:        rkcypb.System_PROCESS,
			Concern:       step.Concern,
			Command:       step.Command,
			Key:           step.Key,
			Payload:       payloadBytes,
			EffectiveTime: timestamppb.New(step.EffectiveTime),
		}
	}

	return prepSteps, nil
}

func ExecuteTxnSync(
	ctx context.Context,
	plat Platform,
	aprod ApecsProducer,
	txn *Txn,
	timeout time.Duration,
	wg *sync.WaitGroup,
) (*ResultProto, error) {
	ctx, span := plat.Telem().StartFunc(ctx)
	defer span.End()

	if aprod.ResponseTarget() == nil {
		err := errors.New("No ResponseTarget in ApecsProducer")
		RecordSpanError(span, err)
		return nil, err
	}

	prepSteps, err := prepareApecsSteps(
		ctx,
		txn,
		plat.Telem(),
		wg,
	)
	if err != nil {
		return nil, err
	}

	respCh := RespChan{
		TxnId:     prepSteps.txnId,
		RespCh:    make(chan *rkcypb.ApecsTxn),
		StartTime: time.Now(),
	}
	defer close(respCh.RespCh)

	aprod.RegisterResponseChannel(&respCh)

	err = executeTxn(
		aprod,
		prepSteps.txnId,
		nil,
		prepSteps.traceParent,
		prepSteps.uponError,
		prepSteps.steps,
		wg,
	)

	if err != nil {
		RecordSpanError(span, err)
		return nil, err
	}

	txnResp, err := waitForResponse(ctx, respCh.RespCh, timeout)
	if err != nil {
		RecordSpanError(span, err)
		return nil, err
	}
	if txnResp == nil {
		err := errors.New("nil txn received")
		RecordSpanError(span, err)
		return nil, err
	}

	success, resProto, result := ApecsTxnResult(ctx, plat, txnResp)
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
		RecordSpanError(span, errProto)
		return nil, errProto
	}

	return resProto, nil
}

func ExecuteTxnAsync(
	ctx context.Context,
	aprod ApecsProducer,
	txn *Txn,
	telem *Telemetry,
	wg *sync.WaitGroup,
) error {
	ctx, span := telem.StartFunc(ctx)
	defer span.End()

	prepSteps, err := prepareApecsSteps(
		ctx,
		txn,
		telem,
		wg,
	)
	if err != nil {
		RecordSpanError(span, err)
		return err
	}

	return executeTxn(
		aprod,
		prepSteps.txnId,
		nil,
		prepSteps.traceParent,
		prepSteps.uponError,
		prepSteps.steps,
		wg,
	)
}

func executeTxn(
	aprod ApecsProducer,
	txnId string,
	assocTxn *rkcypb.AssocTxn,
	traceParent string,
	uponError rkcypb.UponError,
	steps []*rkcypb.ApecsTxn_Step,
	wg *sync.WaitGroup,
) error {
	txn, err := newApecsTxn(txnId, assocTxn, aprod.ResponseTarget(), uponError, steps)
	if err != nil {
		return err
	}

	return produceCurrentStep(aprod, txn, traceParent, wg)
}

var gSystemToTopic = map[rkcypb.System]StandardTopicName{
	rkcypb.System_PROCESS:      PROCESS,
	rkcypb.System_STORAGE:      STORAGE,
	rkcypb.System_STORAGE_SCND: STORAGE_SCND,
}

func produceError(
	ctx context.Context,
	aprod ApecsProducer,
	rtxn *rtApecsTxn,
	step *rkcypb.ApecsTxn_Step,
	code rkcypb.Code,
	logToResult bool,
	msg string,
	wg *sync.WaitGroup,
) error {
	if step == nil {
		step = rtxn.firstForwardStep()
		if step == nil {
			return fmt.Errorf("ApecsKafkaProducer.error TxnId=%s: failed to get firstForwardStep", rtxn.txn.Id)
		}
		msg = "DEFAULTING TO FIRST FORWARD STEP FOR LOGGING!!! - " + msg
	}

	// if no result, put one in so we can log to it
	if step.Result == nil {
		step.Result = &rkcypb.ApecsTxn_Step_Result{
			Code:          code,
			ProcessedTime: timestamppb.Now(),
		}
	}

	if logToResult {
		step.Result.LogEvents = append(
			step.Result.LogEvents,
			&rkcypb.LogEvent{
				Sev: rkcypb.Severity_ERR,
				Msg: msg,
			},
		)
	}

	prd, err := aprod.GetProducer(step.Concern, ERROR, wg)
	if err != nil {
		return err
	}

	txnSer, err := proto.Marshal(rtxn.txn)
	if err != nil {
		return err
	}

	prd.Produce(rkcypb.Directive_APECS_TXN, rtxn.traceParent, []byte(step.Key), txnSer, nil)

	if rtxn.txn.ResponseTarget != nil {
		err = produceResponse(ctx, aprod.Platform(), rtxn.txn.ResponseTarget.Brokers, rtxn, wg)
		if err != nil {
			return err
		}
	}

	return nil
}

func produceComplete(
	ctx context.Context,
	aprod ApecsProducer,
	rtxn *rtApecsTxn,
	wg *sync.WaitGroup,
) error {
	// LORRNOTE 2021-03-21: Complete messages to to the concern of the
	// first step, which I think makes sense in most cases
	step := rtxn.firstForwardStep()
	if step == nil {
		return fmt.Errorf("ApecsKafkaProducer.complete TxnId=%s: failed to get firstForwardStep", rtxn.txn.Id)
	}

	prd, err := aprod.GetProducer(step.Concern, COMPLETE, wg)
	if err != nil {
		return err
	}

	txnSer, err := proto.Marshal(rtxn.txn)
	if err != nil {
		return err
	}

	prd.Produce(rkcypb.Directive_APECS_TXN, rtxn.traceParent, []byte(step.Key), txnSer, nil)

	if rtxn.txn.ResponseTarget != nil {
		err = produceResponse(ctx, aprod.Platform(), rtxn.txn.ResponseTarget.Brokers, rtxn, wg)
		if err != nil {
			return err
		}
	}

	return nil
}

func produceCurrentStep(
	aprod ApecsProducer,
	txn *rkcypb.ApecsTxn,
	traceParent string,
	wg *sync.WaitGroup,
) error {
	rtxn, err := newRtApecsTxn(txn, traceParent)
	if err != nil {
		return err
	}

	step := rtxn.currentStep()

	var prd Producer = nil
	topicName, ok := gSystemToTopic[step.System]
	if !ok {
		return fmt.Errorf("ApecsKafkaProducer.produceCurrentStep TxnId=%s System=%s: Invalid System", rtxn.txn.Id, step.System.String())
	}

	prd, err = aprod.GetProducer(step.Concern, topicName, wg)
	if err != nil {
		return err
	}

	txnSer, err := proto.Marshal(txn)
	if err != nil {
		return err
	}

	var hashKey []byte
	if !isKeylessStep(step) {
		hashKey = []byte(step.Key)
	} else {
		uid, err := uuid.NewRandom() // use a new randomized string
		if err != nil {
			return fmt.Errorf("ApecsKafkaProducer.produceCurrentStep TxnId=%s System=%s: uuid.NewRandom error: %s", rtxn.txn.Id, step.System.String(), err.Error())
		}
		hashKey = uid[:]
	}

	prd.Produce(rkcypb.Directive_APECS_TXN, traceParent, hashKey, txnSer, nil)
	return nil
}
