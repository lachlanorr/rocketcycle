// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/consts"
)

type Step struct {
	ConcernName string
	Command     Command
	Key         string
	Payload     []byte
}

type ApecsProducer struct {
	ctx               context.Context
	bootstrapServers  string
	platformName      string
	producers         map[string]map[consts.StandardTopicName]*Producer
	responseProducers map[string]*kafka.Producer
}

func NewApecsProducer(
	ctx context.Context,
	bootstrapServers string,
	platformName string,
) *ApecsProducer {

	return &ApecsProducer{
		ctx:               ctx,
		bootstrapServers:  bootstrapServers,
		platformName:      platformName,
		producers:         make(map[string]map[consts.StandardTopicName]*Producer),
		responseProducers: make(map[string]*kafka.Producer),
	}
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

func (aprod *ApecsProducer) produceResponse(txn *ApecsTxn) error {
	if txn.ResponseTarget == nil {
		return nil
	}

	rspTgt := txn.ResponseTarget

	txnSer, err := proto.Marshal(txn)
	if err != nil {
		return err
	}

	kProd, ok := aprod.responseProducers[rspTgt.BootstrapServers]
	if !ok {
		var err error
		kProd, err = kafka.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": rspTgt.BootstrapServers,
		})
		if err != nil {
			return err
		}

		aprod.responseProducers[rspTgt.BootstrapServers] = kProd
	}

	kMsg := kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &rspTgt.TopicName,
			Partition: rspTgt.Partition,
		},
		Value:   txnSer,
		Headers: standardHeaders(Directive_APECS_TXN, txn.ReqId),
	}

	err = kProd.Produce(&kMsg, nil)
	if err != nil {
		return err
	}

	return nil
}

func (aprod *ApecsProducer) Close() {
	for _, concernProds := range aprod.producers {
		for _, pdc := range concernProds {
			pdc.Close()
		}
	}

	for _, responseProd := range aprod.responseProducers {
		responseProd.Close()
	}

	aprod.producers = make(map[string]map[consts.StandardTopicName]*Producer)
}

func (aprod *ApecsProducer) ExecuteTxn(
	reqId string,
	rspTgt *ResponseTarget,
	canRevert bool,
	payload *Buffer,
	steps []Step,
) error {
	stepsPb := make([]*ApecsTxn_Step, len(steps))
	for i, step := range steps {
		stepsPb[i] = &ApecsTxn_Step{
			System:      System_PROCESS,
			ConcernName: step.ConcernName,
			Command:     step.Command,
			Key:         step.Key,
		}
	}
	if payload != nil && len(stepsPb) > 0 {
		stepsPb[0].Payload = payload
	}
	return aprod.executeTxn(
		reqId,
		"",
		rspTgt,
		canRevert,
		stepsPb,
	)
}

func (aprod *ApecsProducer) executeTxn(
	reqId string,
	assocReqId string,
	rspTgt *ResponseTarget,
	canRevert bool,
	steps []*ApecsTxn_Step,
) error {
	txn, err := newApecsTxn(reqId, assocReqId, rspTgt, canRevert, steps)
	if err != nil {
		return err
	}

	return aprod.produceCurrentStep(txn)
}

var systemToTopic = map[System]consts.StandardTopicName{
	System_PROCESS: consts.Process,
	System_STORAGE: consts.Storage,
}

func (aprod *ApecsProducer) produceError(rtxn *rtApecsTxn, step *ApecsTxn_Step, code Code, logToResult bool, msg string) error {
	if step == nil {
		step = rtxn.firstForwardStep()
		if step == nil {
			return fmt.Errorf("ApecsProducer.error ReqId=%s: failed to get firstForwardStep", rtxn.txn.ReqId)
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
				Sev: Severity_ERROR,
				Msg: msg,
			},
		)
	}

	prd, err := aprod.getProducer(step.ConcernName, consts.Error)
	if err != nil {
		return err
	}

	txnSer, err := proto.Marshal(rtxn.txn)
	if err != nil {
		return err
	}

	prd.Produce(Directive_APECS_TXN, rtxn.txn.ReqId, []byte(step.Key), txnSer, nil)

	err = aprod.produceResponse(rtxn.txn)
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
		return fmt.Errorf("ApecsProducer.complete ReqId=%s: failed to get firstForwardStep", rtxn.txn.ReqId)
	}

	prd, err := aprod.getProducer(step.ConcernName, consts.Complete)
	if err != nil {
		return err
	}

	txnSer, err := proto.Marshal(rtxn.txn)
	if err != nil {
		return err
	}

	prd.Produce(Directive_APECS_TXN, rtxn.txn.ReqId, []byte(step.Key), txnSer, nil)

	err = aprod.produceResponse(rtxn.txn)
	if err != nil {
		return err
	}

	return nil
}

func (aprod *ApecsProducer) produceCurrentStep(txn *ApecsTxn) error {
	rtxn, err := newRtApecsTxn(txn)
	if err != nil {
		return err
	}

	step := rtxn.currentStep()

	var prd *Producer = nil
	topicName, ok := systemToTopic[step.System]
	if !ok {
		return fmt.Errorf("ApecsProducer.Process ReqId=%s System=%d: Invalid System", rtxn.txn.ReqId, step.System)
	}

	prd, err = aprod.getProducer(step.ConcernName, topicName)
	if err != nil {
		return err
	}

	txnSer, err := proto.Marshal(txn)
	if err != nil {
		return err
	}

	prd.Produce(Directive_APECS_TXN, txn.ReqId, []byte(step.Key), txnSer, nil)
	return nil
}
