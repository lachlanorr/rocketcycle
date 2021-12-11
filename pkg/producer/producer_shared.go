// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package producer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

const DEFAULT_SYNC_TIMEOUT time.Duration = time.Minute * 5

func waitForResponse(ctx context.Context, respCh <-chan *rkcypb.ApecsTxn, timeout time.Duration) (*rkcypb.ApecsTxn, error) {
	if timeout.Seconds() == 0 {
		timeout = DEFAULT_SYNC_TIMEOUT
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
	txn *rkcy.Txn,
	telem *rkcy.Telem,
	wg *sync.WaitGroup,
) (*preparedApecsSteps, error) {
	if err := rkcy.ValidateTxn(txn); err != nil {
		return nil, err
	}

	prepSteps := &preparedApecsSteps{}

	prepSteps.traceParent = rkcy.ExtractTraceParent(ctx)
	if prepSteps.traceParent == "" {
		return nil, fmt.Errorf("No SpanContext present in ctx")
	}
	prepSteps.txnId = rkcy.TraceIdFromTraceParent(prepSteps.traceParent)

	if txn.Revert == rkcy.REVERTABLE {
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
	plat rkcy.Platform,
	aprod rkcy.ApecsProducer,
	txn *rkcy.Txn,
	timeout time.Duration,
	wg *sync.WaitGroup,
) (*rkcy.ResultProto, error) {
	ctx, span := plat.Telem().StartFunc(ctx)
	defer span.End()

	if aprod.ResponseTarget() == nil {
		err := errors.New("No ResponseTarget in ApecsProducer")
		rkcy.RecordSpanError(span, err)
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

	respCh := rkcy.RespChan{
		TxnId:     prepSteps.txnId,
		RespCh:    make(chan *rkcypb.ApecsTxn),
		StartTime: time.Now(),
	}
	defer close(respCh.RespCh)

	aprod.RegisterResponseChannel(&respCh)

	err = ExecuteTxn(
		aprod,
		prepSteps.txnId,
		nil,
		prepSteps.traceParent,
		prepSteps.uponError,
		prepSteps.steps,
		wg,
	)

	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, err
	}

	txnResp, err := waitForResponse(ctx, respCh.RespCh, timeout)
	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, err
	}
	if txnResp == nil {
		err := errors.New("nil txn received")
		rkcy.RecordSpanError(span, err)
		return nil, err
	}

	success, resProto, result := rkcy.ApecsTxnResult(ctx, plat, txnResp)
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
		rkcy.RecordSpanError(span, errProto)
		return nil, errProto
	}

	return resProto, nil
}

func ExecuteTxnAsync(
	ctx context.Context,
	aprod rkcy.ApecsProducer,
	txn *rkcy.Txn,
	telem *rkcy.Telem,
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
		rkcy.RecordSpanError(span, err)
		return err
	}

	return ExecuteTxn(
		aprod,
		prepSteps.txnId,
		nil,
		prepSteps.traceParent,
		prepSteps.uponError,
		prepSteps.steps,
		wg,
	)
}

func ExecuteTxn(
	aprod rkcy.ApecsProducer,
	txnId string,
	assocTxn *rkcypb.AssocTxn,
	traceParent string,
	uponError rkcypb.UponError,
	steps []*rkcypb.ApecsTxn_Step,
	wg *sync.WaitGroup,
) error {
	txn, err := rkcy.NewApecsTxn(txnId, assocTxn, aprod.ResponseTarget(), uponError, steps)
	if err != nil {
		return err
	}

	return ProduceCurrentStep(aprod, txn, traceParent, wg)
}

var gSystemToTopic = map[rkcypb.System]rkcy.StandardTopicName{
	rkcypb.System_PROCESS:      rkcy.PROCESS,
	rkcypb.System_STORAGE:      rkcy.STORAGE,
	rkcypb.System_STORAGE_SCND: rkcy.STORAGE_SCND,
}

func ProduceResponse(
	ctx context.Context,
	plat rkcy.Platform,
	respBrokers string,
	rtxn *rkcy.RtApecsTxn,
	wg *sync.WaitGroup,
) error {
	if rtxn.Txn.ResponseTarget == nil {
		return nil
	}

	respTgt := rtxn.Txn.ResponseTarget

	kMsg, err := rkcy.NewKafkaMessage(&respTgt.Topic, respTgt.Partition, rtxn.Txn, rkcypb.Directive_APECS_TXN, rtxn.TraceParent)
	if err != nil {
		return err
	}

	prodCh := plat.GetProducerCh(ctx, respBrokers, wg)
	prodCh <- kMsg

	return nil
}

func ProduceError(
	ctx context.Context,
	aprod rkcy.ApecsProducer,
	rtxn *rkcy.RtApecsTxn,
	step *rkcypb.ApecsTxn_Step,
	code rkcypb.Code,
	logToResult bool,
	msg string,
	wg *sync.WaitGroup,
) error {
	if step == nil {
		step = rtxn.FirstForwardStep()
		if step == nil {
			return fmt.Errorf("ApecsKafkaProducer.error TxnId=%s: failed to get FirstForwardStep", rtxn.Txn.Id)
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

	prd, err := aprod.GetProducer(step.Concern, rkcy.ERROR, wg)
	if err != nil {
		return err
	}

	txnSer, err := proto.Marshal(rtxn.Txn)
	if err != nil {
		return err
	}

	prd.Produce(rkcypb.Directive_APECS_TXN, rtxn.TraceParent, []byte(step.Key), txnSer, nil)

	if rtxn.Txn.ResponseTarget != nil {
		err = ProduceResponse(ctx, aprod.Platform(), rtxn.Txn.ResponseTarget.Brokers, rtxn, wg)
		if err != nil {
			return err
		}
	}

	return nil
}

func ProduceComplete(
	ctx context.Context,
	aprod rkcy.ApecsProducer,
	rtxn *rkcy.RtApecsTxn,
	wg *sync.WaitGroup,
) error {
	// LORRNOTE 2021-03-21: Complete messages to to the concern of the
	// first step, which I think makes sense in most cases
	step := rtxn.FirstForwardStep()
	if step == nil {
		return fmt.Errorf("ApecsKafkaProducer.complete TxnId=%s: failed to get FirstForwardStep", rtxn.Txn.Id)
	}

	prd, err := aprod.GetProducer(step.Concern, rkcy.COMPLETE, wg)
	if err != nil {
		return err
	}

	txnSer, err := proto.Marshal(rtxn.Txn)
	if err != nil {
		return err
	}

	prd.Produce(rkcypb.Directive_APECS_TXN, rtxn.TraceParent, []byte(step.Key), txnSer, nil)

	if rtxn.Txn.ResponseTarget != nil {
		err = ProduceResponse(ctx, aprod.Platform(), rtxn.Txn.ResponseTarget.Brokers, rtxn, wg)
		if err != nil {
			return err
		}
	}

	return nil
}

func ProduceCurrentStep(
	aprod rkcy.ApecsProducer,
	txn *rkcypb.ApecsTxn,
	traceParent string,
	wg *sync.WaitGroup,
) error {
	rtxn, err := rkcy.NewRtApecsTxn(txn, traceParent)
	if err != nil {
		return err
	}

	step := rtxn.CurrentStep()

	var prod rkcy.Producer = nil
	topicName, ok := gSystemToTopic[step.System]
	if !ok {
		return fmt.Errorf("ApecsKafkaProducer.produceCurrentStep TxnId=%s System=%s: Invalid System", rtxn.Txn.Id, step.System.String())
	}

	prod, err = aprod.GetProducer(step.Concern, topicName, wg)
	if err != nil {
		return err
	}

	txnSer, err := proto.Marshal(txn)
	if err != nil {
		return err
	}

	var hashKey []byte
	if !rkcy.IsKeylessStep(step) {
		hashKey = []byte(step.Key)
	} else {
		uid, err := uuid.NewRandom() // use a new randomized string
		if err != nil {
			return fmt.Errorf("ApecsKafkaProducer.produceCurrentStep TxnId=%s System=%s: uuid.NewRandom error: %s", rtxn.Txn.Id, step.System.String(), err.Error())
		}
		hashKey = uid[:]
	}

	prod.Produce(rkcypb.Directive_APECS_TXN, traceParent, hashKey, txnSer, nil)
	return nil
}
