// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

type OfflinePlatform struct {
	name         string
	environment  string
	telem        *Telemetry
	system       System
	adminBrokers string

	concernHandlers ConcernHandlers
	instanceStore   *InstanceStore
}

func NewOfflinePlatform(
	name string,
	environment string,
) *OfflinePlatform {
	return &OfflinePlatform{
		name:        name,
		environment: environment,
	}
}

func (oplat *OfflinePlatform) Name() string {
	return oplat.name
}

func (oplat *OfflinePlatform) Environment() string {
	return oplat.environment
}

func (oplat *OfflinePlatform) Telem() *Telemetry {
	return oplat.telem
}

func (oplat *OfflinePlatform) System() System {
	return oplat.system
}

func (oplat *OfflinePlatform) SetSystem(system System) {
	oplat.system = system
}

func (oplat *OfflinePlatform) AdminBrokers() string {
	return oplat.adminBrokers
}

func (oplat *OfflinePlatform) UpdateStorageTargets(platMsg *PlatformMessage) {
}

func (oplat *OfflinePlatform) StorageTarget() string {
	return ""
}
func (oplat *OfflinePlatform) ConcernHandlers() ConcernHandlers {
	return oplat.concernHandlers
}
func (oplat *OfflinePlatform) InstanceStore() *InstanceStore {
	return oplat.instanceStore
}

func (oplat *OfflinePlatform) NewApecsProducer(
	ctx context.Context,
	respTarget *TopicTarget,
	wg *sync.WaitGroup,
) ApecsProducer {
	return nil
}

func (oplat *OfflinePlatform) GetProducerCh(
	ctx context.Context,
	brokers string,
	wg *sync.WaitGroup,
) ProducerCh {
	return nil
}

func (oplat *OfflinePlatform) NewProducer(
	ctx context.Context,
	concernName string,
	topicName string,
	wg *sync.WaitGroup,
) Producer {
	return nil
}

func (oplat *OfflinePlatform) AppendCobraCommand(cmd *cobra.Command) {
}

func (oplat *OfflinePlatform) SetStorageInit(name string, storageInit StorageInit) {
}

func (oplat *OfflinePlatform) RegisterLogicHandler(concern string, handler interface{}) {
}

func (oplat *OfflinePlatform) RegisterCrudHandler(storageType string, concern string, handler interface{}) {
}

func (oplat *OfflinePlatform) Start() {
}

type ApecsOfflineProducer struct {
}

func NewApecsOfflineProducer(
	ctx context.Context,
	plat *Platform,
	respTarget *TopicTarget,
	wg *sync.WaitGroup,
) *ApecsOfflineProducer {
	return &ApecsOfflineProducer{}
}

func (aoprod *ApecsOfflineProducer) ResponseTarget() *TopicTarget {
	return nil
}

func (aoprod *ApecsOfflineProducer) ResponseChannel(
	ctx context.Context,
	brokers string,
	wg *sync.WaitGroup,
) ProducerCh {
	return nil
}

func (aoprod *ApecsOfflineProducer) GetProducer(
	concernName string,
	topicName StandardTopicName,
	wg *sync.WaitGroup,
) (Producer, error) {
	return nil, nil
}

func (aoprod *ApecsOfflineProducer) ExecuteTxnSync(
	ctx context.Context,
	txn *Txn,
	timeout time.Duration,
	wg *sync.WaitGroup,
) (*ResultProto, error) {
	return nil, nil
}
