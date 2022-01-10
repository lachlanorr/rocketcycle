// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/config_mgr"
	"github.com/lachlanorr/rocketcycle/pkg/mgmt"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/pkg/stream"
)

type RespChan struct {
	TxnId     string
	RespCh    chan *rkcypb.ApecsTxn
	StartTime time.Time
}

type Platform struct {
	ctx context.Context
	wg  *sync.WaitGroup

	Args *rkcy.PlatformArgs

	configMgr *config_mgr.ConfigMgr
	configRdr rkcy.ConfigRdr

	bprod         *stream.BrokersProducer
	ApecsProducer *stream.ApecsProducer

	RtPlatformDef *rkcy.RtPlatformDef

	InstanceStore *rkcy.InstanceStore

	ClientCode *rkcy.ClientCode

	respTarget     *rkcypb.TopicTarget
	respRegisterCh chan *RespChan
}

func NewPlatform(
	ctx context.Context,
	wg *sync.WaitGroup,
	id string,
	platform string,
	environment string,
	adminBrokers string,
	adminPingInterval time.Duration,
	strmprov rkcy.StreamProvider,
	clientCode *rkcy.ClientCode,
	respTarget *rkcypb.TopicTarget,
) (*Platform, error) {
	if !rkcy.IsValidName(platform) {
		return nil, fmt.Errorf("Invalid platform: %s", platform)
	}

	plat := &Platform{
		ctx: ctx,
		wg:  wg,
		Args: &rkcy.PlatformArgs{
			Id:                id,
			Platform:          platform,
			Environment:       environment,
			AdminBrokers:      adminBrokers,
			AdminPingInterval: adminPingInterval,
		},

		InstanceStore: rkcy.NewInstanceStore(),

		ClientCode: clientCode,

		respTarget: respTarget,
	}

	plat.bprod = stream.NewBrokersProducer(strmprov)
	plat.ApecsProducer = stream.NewApecsProducer(
		ctx,
		wg,
		plat.Args,
		plat.bprod,
	)

	if respTarget != nil {
		plat.respRegisterCh = make(chan *RespChan, 10)

		wg.Add(1)
		go ConsumeResponseTopic(ctx, wg, plat.bprod.StreamProvider, plat.respTarget, plat.respRegisterCh)
	}

	// cleanup when context closes
	plat.wg.Add(1)
	go func() {
		select {
		case <-ctx.Done():
			log.Info().
				Str("Platform", plat.Args.Platform).
				Str("Environment", plat.Args.Environment).
				Msgf("Platform closed %s", plat.Args.Id)
			plat.wg.Done()
		}
	}()

	return plat, nil
}

func (plat *Platform) Context() context.Context {
	return plat.ctx
}

func (plat *Platform) WaitGroup() *sync.WaitGroup {
	return plat.wg
}

func (plat *Platform) StreamProvider() rkcy.StreamProvider {
	return plat.bprod.StreamProvider
}

func (plat *Platform) ResponseTarget() *rkcypb.TopicTarget {
	return plat.respTarget
}

func (plat *Platform) RegisterResponseChannel(respCh *RespChan) {
	if plat.respRegisterCh == nil {
		log.Error().Msgf("RegisterResponseChannel called with no ResponseTarget")
		return
	}
	plat.respRegisterCh <- respCh
}

func (plat *Platform) GetProducerCh(
	ctx context.Context,
	wg *sync.WaitGroup,
	brokers string,
) rkcy.ProducerCh {
	return plat.bprod.GetProducerCh(ctx, wg, brokers)
}

func (plat *Platform) NewManagedProducer(
	ctx context.Context,
	wg *sync.WaitGroup,
	concernName string,
	topicName string,
) *stream.ManagedProducer {
	return stream.NewManagedProducer(
		ctx,
		wg,
		plat.Args,
		plat.bprod,
		concernName,
		topicName,
	)
}

func (plat *Platform) ConfigMgr() *config_mgr.ConfigMgr {
	if plat.configMgr == nil {
		log.Fatal().Msg("Platform.ConfigMgr(): InitConfigMgr has not been called")
	}
	return plat.configMgr
}

func (plat *Platform) ConfigRdr() rkcy.ConfigRdr {
	if plat.configRdr == nil {
		log.Fatal().Msg("Platform.ConfigRdr(): InitConfigMgr has not been called")
	}
	return plat.configRdr
}

func (plat *Platform) InitConfigMgr(ctx context.Context) {
	if plat.configMgr != nil {
		log.Fatal().Msg("InitConfigMgr called twice")
	}
	plat.configMgr = config_mgr.NewConfigMgr(
		ctx,
		plat.WaitGroup(),
		plat.StreamProvider(),
		plat.Args.Platform,
		plat.Args.Environment,
		plat.Args.AdminBrokers,
	)
	plat.configRdr = rkcy.ConfigRdr(config_mgr.NewConfigMgrRdr(plat.configMgr))
}

func (plat *Platform) UpdateStorageTargets(platMsg *mgmt.PlatformMessage) {
	if (platMsg.Directive & rkcypb.Directive_PLATFORM) != rkcypb.Directive_PLATFORM {
		log.Error().Msgf("Invalid directive for PlatformTopic: %s", platMsg.Directive.String())
		return
	}
	plat.ClientCode.UpdateStorageTargets(platMsg.NewRtPlatDef.StorageTargets)
}

func respondThroughChannel(txnId string, respCh chan<- *rkcypb.ApecsTxn, txn *rkcypb.ApecsTxn) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Str("TxnId", txnId).
				Msgf("recover while sending to respCh '%s'", r)
		}
	}()
	respCh <- txn
}

func ConsumeResponseTopic(
	ctx context.Context,
	wg *sync.WaitGroup,
	strmprov rkcy.StreamProvider,
	respTarget *rkcypb.TopicTarget,
	respRegisterCh <-chan *RespChan,
) {
	defer wg.Done()

	reqMap := make(map[string]*RespChan)

	groupName := fmt.Sprintf("rkcy_response_%s", respTarget.Topic)

	kafkaLogCh := make(chan kafka.LogEvent)
	go rkcy.PrintKafkaLogs(ctx, kafkaLogCh)

	cons, err := strmprov.NewConsumer(respTarget.Brokers, groupName, kafkaLogCh)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("BootstrapServers", respTarget.Brokers).
			Str("GroupId", groupName).
			Msg("Unable to plat.NewConsumer")
	}
	shouldCommit := false

	defer func() {
		log.Trace().
			Str("Topic", respTarget.Topic).
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
		err := cons.Close()
		if err != nil {
			log.Error().
				Err(err).
				Str("Topic", respTarget.Topic).
				Msgf("Error during consumer.Close()")
		}
		log.Trace().
			Str("Topic", respTarget.Topic).
			Msgf("CONSUMER CLOSED")
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
			log.Trace().
				Msg("ConsumeResponseTopic exiting, ctx.Done()")
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
				case rch := <-respRegisterCh:
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
					log.Debug().
						Str("Topic", respTarget.Topic).
						Int32("Partition", respTarget.Partition).
						Int64("Offset", int64(msg.TopicPartition.Offset)).
						Msgf("Initial offset committed: %+v", comOffs)
				}

				directive := rkcy.GetDirective(msg)
				if directive == rkcypb.Directive_APECS_TXN {
					txnId := rkcy.GetTraceId(msg)
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
