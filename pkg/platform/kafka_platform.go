// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/config"
	"github.com/lachlanorr/rocketcycle/pkg/consumer"
	"github.com/lachlanorr/rocketcycle/pkg/producer"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type KafkaPlatform struct {
	name         string
	environment  string
	storageInits map[string]rkcy.StorageInit

	adminBrokers      string
	adminPingInterval time.Duration
	storageTarget     string
	telem             *rkcy.Telem

	system    rkcypb.System
	configMgr *config.ConfigMgr
	configRdr rkcy.ConfigRdr

	concernHandlers rkcy.ConcernHandlers
	bprod           *producer.BrokersProducer

	rtPlatDef *rkcy.RtPlatformDef

	instanceStore *rkcy.InstanceStore
}

func NewKafkaPlatform(
	name string,
	environment string,
) (*KafkaPlatform, error) {
	if !rkcy.IsValidName(name) {
		return nil, fmt.Errorf("Invalid name: %s", name)
	}
	environment = os.Getenv("RKCY_ENVIRONMENT")
	if !rkcy.IsValidName(environment) {
		return nil, fmt.Errorf("Invalid RKCY_ENVIRONMENT: %s", environment)
	}

	environment = os.Getenv("RKCY_ENVIRONMENT")
	if !rkcy.IsValidName(environment) {
		return nil, fmt.Errorf("Invalid RKCY_ENVIRONMENT: %s", environment)
	}

	plat := &KafkaPlatform{
		name:         name,
		environment:  environment,
		storageInits: make(map[string]rkcy.StorageInit),

		concernHandlers: gConcernHandlerRegistry,

		instanceStore: rkcy.NewInstanceStore(),
	}

	return plat, nil
}

func (kplat *KafkaPlatform) Init(
	adminBrokers string,
	adminPingInterval time.Duration,
	storageTarget string,
	otelcolEndpoint string,
) error {
	kplat.adminBrokers = adminBrokers
	kplat.adminPingInterval = adminPingInterval
	kplat.storageTarget = storageTarget

	if otelcolEndpoint != "" {
		var err error
		kplat.telem, err = rkcy.NewTelem(context.Background(), otelcolEndpoint)
		if err != nil {
			return err
		}
	}
	kplat.bprod = producer.NewBrokersProducer(kplat)
	return nil
}

func (kplat *KafkaPlatform) PlatformDef() *rkcy.RtPlatformDef {
	return kplat.rtPlatDef
}

func (kplat *KafkaPlatform) SetPlatformDef(rtPlatDef *rkcy.RtPlatformDef) {
	kplat.rtPlatDef = rtPlatDef
}

func (kplat *KafkaPlatform) AdminPingInterval() time.Duration {
	return kplat.adminPingInterval
}

func (kplat *KafkaPlatform) Name() string {
	return kplat.name
}

func (kplat *KafkaPlatform) Environment() string {
	return kplat.environment
}

func (kplat *KafkaPlatform) Telem() *rkcy.Telem {
	return kplat.telem
}

func (kplat *KafkaPlatform) System() rkcypb.System {
	return kplat.system
}

func (kplat *KafkaPlatform) SetSystem(system rkcypb.System) {
	kplat.system = system
}

func (kplat *KafkaPlatform) AdminBrokers() string {
	return kplat.adminBrokers
}

func (kplat *KafkaPlatform) StorageTarget() string {
	return kplat.storageTarget
}

func (kplat *KafkaPlatform) InstanceStore() *rkcy.InstanceStore {
	return kplat.instanceStore
}

func (kplat *KafkaPlatform) ConcernHandlers() rkcy.ConcernHandlers {
	return kplat.concernHandlers
}

func (kplat *KafkaPlatform) SetStorageInit(name string, storageInit rkcy.StorageInit) {
	kplat.storageInits[name] = storageInit
}

func (kplat *KafkaPlatform) NewProducer(bootstrapServers string, logCh chan kafka.LogEvent) (rkcy.Producer, error) {
	return producer.NewKafkaProducer(bootstrapServers, logCh)
}

func (kplat *KafkaPlatform) NewManagedProducer(
	ctx context.Context,
	concernName string,
	topicName string,
	wg *sync.WaitGroup,
) rkcy.ManagedProducer {
	pdc := producer.NewKafkaManagedProducer(
		ctx,
		rkcy.Platform(kplat),
		concernName,
		topicName,
		wg,
	)

	return rkcy.ManagedProducer(pdc)
}

func (kplat *KafkaPlatform) NewApecsProducer(
	ctx context.Context,
	respTarget *rkcypb.TopicTarget,
	wg *sync.WaitGroup,
) rkcy.ApecsProducer {
	kprod := producer.NewKafkaApecsProducer(ctx, kplat, respTarget, wg)
	return rkcy.ApecsProducer(kprod)
}

func (kplat *KafkaPlatform) GetProducerCh(
	ctx context.Context,
	brokers string,
	wg *sync.WaitGroup,
) rkcy.ProducerCh {
	return kplat.bprod.GetProducerCh(ctx, brokers, wg)
}

func (*KafkaPlatform) NewConsumer(bootstrapServers string, groupName string, logCh chan kafka.LogEvent) (rkcy.Consumer, error) {
	kcons, err := consumer.NewKafkaConsumer(bootstrapServers, groupName, logCh)
	if err != nil {
		return nil, err
	}
	return rkcy.Consumer(kcons), nil
}

func (kplat *KafkaPlatform) RegisterLogicHandler(concern string, handler interface{}) {
	kplat.concernHandlers.RegisterLogicHandler(concern, handler)
}

func (kplat *KafkaPlatform) RegisterCrudHandler(storageType string, concern string, handler interface{}) {
	kplat.concernHandlers.RegisterCrudHandler(storageType, concern, handler)
}

func (kplat *KafkaPlatform) ConfigMgr() *config.ConfigMgr {
	if kplat.configMgr == nil {
		log.Fatal().Msg("InitConfigMgr has not been called")
	}
	return kplat.configMgr
}

func (kplat *KafkaPlatform) ConfigRdr() rkcy.ConfigRdr {
	if kplat.configRdr == nil {
		log.Fatal().Msg("InitConfigMgr has not been called")
	}
	return kplat.configRdr
}

func (kplat *KafkaPlatform) InitConfigMgr(ctx context.Context, wg *sync.WaitGroup) {
	if kplat.configMgr != nil {
		log.Fatal().Msg("InitConfigMgr called twice")
	}
	kplat.configMgr = config.NewConfigMgr(
		ctx,
		rkcy.Platform(kplat),
		wg,
	)
	kplat.configRdr = rkcy.ConfigRdr(config.NewConfigMgrRdr(kplat.configMgr))
}

func (kplat *KafkaPlatform) UpdateStorageTargets(platMsg *rkcy.PlatformMessage) {
	if (platMsg.Directive & rkcypb.Directive_PLATFORM) != rkcypb.Directive_PLATFORM {
		log.Error().Msgf("Invalid directive for PlatformTopic: %s", platMsg.Directive.String())
		return
	}

	if rkcy.IsStorageSystem(kplat.system) {
		stgTgtInits := make(map[string]*rkcy.StorageTargetInit)
		for _, platStgTgt := range platMsg.NewRtPlatDef.StorageTargets {
			tgt := &rkcy.StorageTargetInit{
				StorageTarget: platStgTgt,
			}
			if tgt.Config == nil {
				tgt.Config = make(map[string]string)
			}
			var ok bool
			tgt.Init, ok = kplat.storageInits[platStgTgt.Type]
			if !ok {
				log.Warn().
					Str("StorageTarget", platStgTgt.Name).
					Msgf("No StorageInit for target")
			}
			stgTgtInits[platStgTgt.Name] = tgt
		}
		for _, cncHdlr := range kplat.concernHandlers {
			cncHdlr.SetStorageTargets(stgTgtInits)
		}
	}
}
