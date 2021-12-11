// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package producer

import (
	"context"
	"hash"
	"hash/fnv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/consumer"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type KafkaProducer struct {
	bprod *BrokersProducer

	id string

	constlog zerolog.Logger
	slog     zerolog.Logger

	platformName string
	concernName  string
	concern      *rkcy.RtConcern
	topicName    string

	adminPingInterval time.Duration

	brokers string
	prodCh  rkcy.ProducerCh
	topics  *rkcy.RtTopics

	producersTopic string
	adminProdCh    rkcy.ProducerCh

	doneCh     chan struct{}
	pauseCh    chan bool
	platformCh chan *rkcy.PlatformMessage
	produceCh  chan *message

	fnv64 hash.Hash64
}

type message struct {
	directive   rkcypb.Directive
	traceParent string
	key         []byte
	value       []byte
	deliveryCh  chan kafka.Event
}

func NewKafkaProducer(
	ctx context.Context,
	bprod *BrokersProducer,
	adminBrokers string,
	platformName string,
	environment string,
	concernName string,
	topicName string,
	adminPingInterval time.Duration,
	wg *sync.WaitGroup,
) *KafkaProducer {
	prod := KafkaProducer{
		bprod: bprod,
		id:    uuid.NewString(),
		constlog: log.With().
			Str("Concern", concernName).
			Logger(),
		platformName:      platformName,
		concernName:       concernName,
		topicName:         topicName,
		adminPingInterval: adminPingInterval,
		fnv64:             fnv.New64(),
	}

	prod.slog = prod.constlog.With().Logger()
	prod.doneCh = make(chan struct{})
	prod.pauseCh = make(chan bool)
	prod.platformCh = make(chan *rkcy.PlatformMessage)
	prod.produceCh = make(chan *message)

	prod.producersTopic = rkcy.ProducersTopic(platformName, environment)
	prod.adminProdCh = prod.bprod.GetProducerCh(ctx, adminBrokers, wg)

	consumer.ConsumePlatformTopic(
		ctx,
		prod.platformCh,
		adminBrokers,
		platformName,
		environment,
		nil,
		wg,
	)

	platMsg := <-prod.platformCh
	prod.updatePlatform(ctx, platMsg.NewRtPlatDef, wg)

	go prod.run(ctx, wg)

	return &prod
}

func (krprod *KafkaProducer) updatePlatform(
	ctx context.Context,
	rtPlatDef *rkcy.RtPlatformDef,
	wg *sync.WaitGroup,
) {
	var ok bool
	krprod.concern, ok = rtPlatDef.Concerns[krprod.concernName]
	if !ok {
		krprod.slog.Error().
			Msg("updatePlatform: Failed to find Concern")
		return
	}

	krprod.topics, ok = krprod.concern.Topics[krprod.topicName]
	if !ok {
		krprod.slog.Error().
			Msg("updatePlatform: Failed to find Topics")
		return
	}

	// reset to copy of constlog since we will replace "Topic" sometimes
	krprod.slog = krprod.constlog.With().Str("Topic", krprod.topics.CurrentTopic).Logger()

	// update producer if necessary
	if krprod.brokers != krprod.topics.CurrentCluster.Brokers {
		krprod.brokers = krprod.topics.CurrentCluster.Brokers
		krprod.prodCh = krprod.bprod.GetProducerCh(ctx, krprod.brokers, wg)
		krprod.slog = krprod.slog.With().
			Str("Brokers", krprod.brokers).
			Logger()
	}
}

func (krprod *KafkaProducer) producerDirective() *rkcypb.ProducerDirective {
	return &rkcypb.ProducerDirective{
		Id:          krprod.id,
		ConcernName: krprod.concernName,
		ConcernType: krprod.concern.Concern.Type,
		Topic:       krprod.topicName,
		Generation:  krprod.topics.Topics.Current.Generation,
	}
}

func (krprod *KafkaProducer) Produce(
	directive rkcypb.Directive,
	traceParent string,
	key []byte,
	value []byte,
	deliveryChan chan kafka.Event,
) {
	krprod.produceCh <- &message{
		directive:   directive,
		traceParent: traceParent,
		key:         key,
		value:       value,
		deliveryCh:  deliveryChan,
	}
}

func (krprod *KafkaProducer) Close() {
	krprod.doneCh <- struct{}{}
}

func (krprod *KafkaProducer) run(ctx context.Context, wg *sync.WaitGroup) {
	pingAdminTicker := time.NewTicker(krprod.adminPingInterval)
	pingMsg, err := rkcy.NewKafkaMessage(
		&krprod.producersTopic,
		0,
		krprod.producerDirective(),
		rkcypb.Directive_PRODUCER_STATUS,
		"",
	)
	if err != nil {
		krprod.slog.Fatal().
			Err(err).
			Msg("Failed to create pingMsg")
	}

	paused := false
	for {
		if !paused {
			select {
			case <-ctx.Done():
				return
			case <-krprod.doneCh:
				return
			case <-pingAdminTicker.C:
				krprod.adminProdCh <- pingMsg
			case paused = <-krprod.pauseCh:
				var directive rkcypb.Directive
				if paused {
					directive = rkcypb.Directive_PRODUCER_PAUSED
				} else {
					directive = rkcypb.Directive_PRODUCER_RUNNING
				}
				msg, err := rkcy.NewKafkaMessage(
					&krprod.producersTopic,
					0,
					krprod.producerDirective(),
					directive,
					"",
				)
				if err != nil {
					krprod.slog.Error().
						Err(err).
						Msg("Failed to kafkaMessage")
					continue
				}
				log.Info().Msgf("pause produce %+v", msg)
				krprod.adminProdCh <- msg
			case platformMsg := <-krprod.platformCh:
				krprod.updatePlatform(ctx, platformMsg.NewRtPlatDef, wg)
			case msg := <-krprod.produceCh:
				if krprod.prodCh == nil {
					krprod.slog.Error().
						Msg("Failed to Produce, kafka Producer is nil")
					continue
				}

				krprod.fnv64.Reset()
				krprod.fnv64.Write(msg.key)
				fnvCalc := krprod.fnv64.Sum64()
				partition := int32(fnvCalc % uint64(krprod.topics.Topics.Current.PartitionCount))

				kMsg := &kafka.Message{
					TopicPartition: kafka.TopicPartition{
						Topic:     &krprod.topics.CurrentTopic,
						Partition: partition,
					},
					Value:   msg.value,
					Headers: rkcy.StandardHeaders(msg.directive, msg.traceParent),
				}

				krprod.prodCh <- kMsg
			}
		} else {
			// Same select as above without the produceCh read
			// This way we can hang in this bottom select until unpaused
			// and not read from produceCh, and producers will be paused
			// trying to publish to the channel as well.
			select {
			case <-ctx.Done():
				return
			case <-krprod.doneCh:
				return
			case paused = <-krprod.pauseCh:
				continue
			case platMsg := <-krprod.platformCh:
				krprod.updatePlatform(ctx, platMsg.NewRtPlatDef, wg)
			}
		}
	}
}
