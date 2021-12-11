// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"context"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type OfflinePlatform struct {
	name              string
	environment       string
	telem             *rkcy.Telem
	system            rkcypb.System
	adminBrokers      string
	adminPingInterval time.Duration

	concernHandlers rkcy.ConcernHandlers
	instanceStore   *rkcy.InstanceStore
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

func (oplat *OfflinePlatform) Init(
	adminBrokers string,
	adminPingInterval time.Duration,
	storageTarget string,
	otelcolEndpoint string,
) error {
	oplat.adminBrokers = adminBrokers
	oplat.adminPingInterval = adminPingInterval

	// LORRTODO: set telem to a mock telem interface
	return nil
}

func (oplat *OfflinePlatform) Name() string {
	return oplat.name
}

func (oplat *OfflinePlatform) Environment() string {
	return oplat.environment
}

func (oplat *OfflinePlatform) Telem() *rkcy.Telem {
	return oplat.telem
}

func (oplat *OfflinePlatform) System() rkcypb.System {
	return oplat.system
}

func (oplat *OfflinePlatform) SetSystem(system rkcypb.System) {
	oplat.system = system
}

func (oplat *OfflinePlatform) AdminBrokers() string {
	return oplat.adminBrokers
}

func (oplat *OfflinePlatform) AdminPingInterval() time.Duration {
	return oplat.adminPingInterval
}

func (oplat *OfflinePlatform) UpdateStorageTargets(platMsg *rkcy.PlatformMessage) {
}

func (oplat *OfflinePlatform) StorageTarget() string {
	return ""
}

func (oplat *OfflinePlatform) ConcernHandlers() rkcy.ConcernHandlers {
	return oplat.concernHandlers
}

func (oplat *OfflinePlatform) InstanceStore() *rkcy.InstanceStore {
	return oplat.instanceStore
}

func (oplat *OfflinePlatform) PlatformDef() *rkcy.RtPlatformDef {
	return nil
}

func (oplat *OfflinePlatform) SetPlatformDef(rtPlatDef *rkcy.RtPlatformDef) {
}

func (oplat *OfflinePlatform) ConfigRdr() rkcy.ConfigRdr {
	return nil
}

func (oplat *OfflinePlatform) InitConfigMgr(ctx context.Context, wg *sync.WaitGroup) {
	// no op?
}

func (oplat *OfflinePlatform) NewApecsProducer(
	ctx context.Context,
	respTarget *rkcypb.TopicTarget,
	wg *sync.WaitGroup,
) rkcy.ApecsProducer {
	return nil
}

func (oplat *OfflinePlatform) GetProducerCh(
	ctx context.Context,
	brokers string,
	wg *sync.WaitGroup,
) rkcy.ProducerCh {
	return nil
}

func (oplat *OfflinePlatform) NewProducer(
	ctx context.Context,
	concernName string,
	topicName string,
	wg *sync.WaitGroup,
) rkcy.Producer {
	return nil
}

func (oplat *OfflinePlatform) AppendCobraCommand(cmd *cobra.Command) {
}

func (oplat *OfflinePlatform) SetStorageInit(name string, storageInit rkcy.StorageInit) {
}

func (oplat *OfflinePlatform) RegisterLogicHandler(concern string, handler interface{}) {
}

func (oplat *OfflinePlatform) RegisterCrudHandler(storageType string, concern string, handler interface{}) {
}
