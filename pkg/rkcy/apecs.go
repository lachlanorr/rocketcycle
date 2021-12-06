// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"sync"

	"github.com/spf13/cobra"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type Producer interface {
	Produce(
		directive Directive,
		traceParent string,
		key []byte,
		value []byte,
		deliveryChan chan kafka.Event,
	)
	Close()
}

type ApecsProducer interface {
	Platform() Platform
	ResponseTarget() *TopicTarget
	RegisterResponseChannel(respCh *RespChan)

	GetProducer(
		concernName string,
		topicName StandardTopicName,
		wg *sync.WaitGroup,
	) (Producer, error)
}

type Platform interface {
	Name() string
	Environment() string

	Telem() *Telemetry

	System() System
	SetSystem(system System)

	AdminBrokers() string
	UpdateStorageTargets(platMsg *PlatformMessage)
	StorageTarget() string
	ConcernHandlers() ConcernHandlers
	InstanceStore() *InstanceStore

	NewApecsProducer(
		ctx context.Context,
		respTarget *TopicTarget,
		wg *sync.WaitGroup,
	) ApecsProducer

	GetProducerCh(
		ctx context.Context,
		brokers string,
		wg *sync.WaitGroup,
	) ProducerCh

	NewProducer(
		ctx context.Context,
		concernName string,
		topicName string,
		wg *sync.WaitGroup,
	) Producer

	AppendCobraCommand(cmd *cobra.Command)
	SetStorageInit(name string, storageInit StorageInit)
	RegisterLogicHandler(concern string, handler interface{})
	RegisterCrudHandler(storageType string, concern string, handler interface{})
	Start()
}
