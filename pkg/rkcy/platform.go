// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"sync"
	"time"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type PlatformMessage struct {
	Directive    rkcypb.Directive
	Timestamp    time.Time
	Offset       int64
	NewRtPlatDef *RtPlatformDef
	OldRtPlatDef *RtPlatformDef
}

type ProducerCh chan *kafka.Message

type Platform interface {
	Init(
		adminBrokers string,
		adminPingInterval time.Duration,
		storageTarget string,
		otelcolEndpoint string,
	) error
	InitConfigMgr(ctx context.Context, wg *sync.WaitGroup)

	Name() string
	Environment() string

	Telem() *Telem

	AdminBrokers() string
	AdminPingInterval() time.Duration
	StorageTarget() string

	System() rkcypb.System
	SetSystem(system rkcypb.System)

	UpdateStorageTargets(platMsg *PlatformMessage)
	ConcernHandlers() ConcernHandlers
	InstanceStore() *InstanceStore

	PlatformDef() *RtPlatformDef
	SetPlatformDef(rtPlatDef *RtPlatformDef)
	ConfigRdr() ConfigRdr

	NewProducer(boostrapServers string, logCh chan kafka.LogEvent) (Producer, error)

	NewManagedProducer(
		ctx context.Context,
		concernName string,
		topicName string,
		wg *sync.WaitGroup,
	) ManagedProducer

	NewApecsProducer(
		ctx context.Context,
		respTarget *rkcypb.TopicTarget,
		wg *sync.WaitGroup,
	) ApecsProducer

	GetProducerCh(
		ctx context.Context,
		brokers string,
		wg *sync.WaitGroup,
	) ProducerCh

	NewConsumer(bootstrapServers string, groupName string, logCh chan kafka.LogEvent) (Consumer, error)

	SetStorageInit(name string, storageInit StorageInit)
	RegisterLogicHandler(concern string, handler interface{})
	RegisterCrudHandler(storageType string, concern string, handler interface{})
}
