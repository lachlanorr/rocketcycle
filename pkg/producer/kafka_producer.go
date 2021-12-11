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
	plat rkcy.Platform

	id string

	constlog zerolog.Logger
	slog     zerolog.Logger

	concernName string
	concern     *rkcy.RtConcern
	topicName   string

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
	plat rkcy.Platform,
	concernName string,
	topicName string,
	wg *sync.WaitGroup,
) *KafkaProducer {
	kprod := KafkaProducer{
		plat: plat,
		id:   uuid.NewString(),
		constlog: log.With().
			Str("Concern", concernName).
			Logger(),
		concernName: concernName,
		topicName:   topicName,
		fnv64:       fnv.New64(),
	}

	kprod.slog = kprod.constlog.With().Logger()
	kprod.doneCh = make(chan struct{})
	kprod.pauseCh = make(chan bool)
	kprod.platformCh = make(chan *rkcy.PlatformMessage)
	kprod.produceCh = make(chan *message)

	kprod.producersTopic = rkcy.ProducersTopic(plat.Name(), plat.Environment())
	kprod.adminProdCh = kprod.plat.GetProducerCh(ctx, plat.AdminBrokers(), wg)

	consumer.ConsumePlatformTopic(
		ctx,
		plat,
		kprod.platformCh,
		nil,
		wg,
	)

	platMsg := <-kprod.platformCh
	kprod.updatePlatform(ctx, platMsg.NewRtPlatDef, wg)

	go kprod.run(ctx, wg)

	return &kprod
}

func (kprod *KafkaProducer) updatePlatform(
	ctx context.Context,
	rtPlatDef *rkcy.RtPlatformDef,
	wg *sync.WaitGroup,
) {
	var ok bool
	kprod.concern, ok = rtPlatDef.Concerns[kprod.concernName]
	if !ok {
		kprod.slog.Error().
			Msg("updatePlatform: Failed to find Concern")
		return
	}

	kprod.topics, ok = kprod.concern.Topics[kprod.topicName]
	if !ok {
		kprod.slog.Error().
			Msg("updatePlatform: Failed to find Topics")
		return
	}

	// reset to copy of constlog since we will replace "Topic" sometimes
	kprod.slog = kprod.constlog.With().Str("Topic", kprod.topics.CurrentTopic).Logger()

	// update producer if necessary
	if kprod.brokers != kprod.topics.CurrentCluster.Brokers {
		kprod.brokers = kprod.topics.CurrentCluster.Brokers
		kprod.prodCh = kprod.plat.GetProducerCh(ctx, kprod.brokers, wg)
		kprod.slog = kprod.slog.With().
			Str("Brokers", kprod.brokers).
			Logger()
	}
}

func (kprod *KafkaProducer) producerDirective() *rkcypb.ProducerDirective {
	return &rkcypb.ProducerDirective{
		Id:          kprod.id,
		ConcernName: kprod.concernName,
		ConcernType: kprod.concern.Concern.Type,
		Topic:       kprod.topicName,
		Generation:  kprod.topics.Topics.Current.Generation,
	}
}

func (kprod *KafkaProducer) Produce(
	directive rkcypb.Directive,
	traceParent string,
	key []byte,
	value []byte,
	deliveryChan chan kafka.Event,
) {
	kprod.produceCh <- &message{
		directive:   directive,
		traceParent: traceParent,
		key:         key,
		value:       value,
		deliveryCh:  deliveryChan,
	}
}

func (kprod *KafkaProducer) Close() {
	kprod.doneCh <- struct{}{}
}

func (kprod *KafkaProducer) run(ctx context.Context, wg *sync.WaitGroup) {
	pingAdminTicker := time.NewTicker(kprod.plat.AdminPingInterval())
	pingMsg, err := rkcy.NewKafkaMessage(
		&kprod.producersTopic,
		0,
		kprod.producerDirective(),
		rkcypb.Directive_PRODUCER_STATUS,
		"",
	)
	if err != nil {
		kprod.slog.Fatal().
			Err(err).
			Msg("Failed to create pingMsg")
	}

	paused := false
	for {
		if !paused {
			select {
			case <-ctx.Done():
				return
			case <-kprod.doneCh:
				return
			case <-pingAdminTicker.C:
				kprod.adminProdCh <- pingMsg
			case paused = <-kprod.pauseCh:
				var directive rkcypb.Directive
				if paused {
					directive = rkcypb.Directive_PRODUCER_PAUSED
				} else {
					directive = rkcypb.Directive_PRODUCER_RUNNING
				}
				msg, err := rkcy.NewKafkaMessage(
					&kprod.producersTopic,
					0,
					kprod.producerDirective(),
					directive,
					"",
				)
				if err != nil {
					kprod.slog.Error().
						Err(err).
						Msg("Failed to kafkaMessage")
					continue
				}
				log.Info().Msgf("pause produce %+v", msg)
				kprod.adminProdCh <- msg
			case platformMsg := <-kprod.platformCh:
				kprod.updatePlatform(ctx, platformMsg.NewRtPlatDef, wg)
			case msg := <-kprod.produceCh:
				if kprod.prodCh == nil {
					kprod.slog.Error().
						Msg("Failed to Produce, kafka Producer is nil")
					continue
				}

				kprod.fnv64.Reset()
				kprod.fnv64.Write(msg.key)
				fnvCalc := kprod.fnv64.Sum64()
				partition := int32(fnvCalc % uint64(kprod.topics.Topics.Current.PartitionCount))

				kMsg := &kafka.Message{
					TopicPartition: kafka.TopicPartition{
						Topic:     &kprod.topics.CurrentTopic,
						Partition: partition,
					},
					Value:   msg.value,
					Headers: rkcy.StandardHeaders(msg.directive, msg.traceParent),
				}

				kprod.prodCh <- kMsg
			}
		} else {
			// Same select as above without the produceCh read
			// This way we can hang in this bottom select until unpaused
			// and not read from produceCh, and producers will be paused
			// trying to publish to the channel as well.
			select {
			case <-ctx.Done():
				return
			case <-kprod.doneCh:
				return
			case paused = <-kprod.pauseCh:
				continue
			case platMsg := <-kprod.platformCh:
				kprod.updatePlatform(ctx, platMsg.NewRtPlatDef, wg)
			}
		}
	}
}
