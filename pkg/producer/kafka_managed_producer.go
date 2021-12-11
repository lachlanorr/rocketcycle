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

type KafkaManagedProducer struct {
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

func NewKafkaManagedProducer(
	ctx context.Context,
	plat rkcy.Platform,
	concernName string,
	topicName string,
	wg *sync.WaitGroup,
) *KafkaManagedProducer {
	kprod := KafkaManagedProducer{
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

func (kmprod *KafkaManagedProducer) updatePlatform(
	ctx context.Context,
	rtPlatDef *rkcy.RtPlatformDef,
	wg *sync.WaitGroup,
) {
	var ok bool
	kmprod.concern, ok = rtPlatDef.Concerns[kmprod.concernName]
	if !ok {
		kmprod.slog.Error().
			Msg("updatePlatform: Failed to find Concern")
		return
	}

	kmprod.topics, ok = kmprod.concern.Topics[kmprod.topicName]
	if !ok {
		kmprod.slog.Error().
			Msg("updatePlatform: Failed to find Topics")
		return
	}

	// reset to copy of constlog since we will replace "Topic" sometimes
	kmprod.slog = kmprod.constlog.With().Str("Topic", kmprod.topics.CurrentTopic).Logger()

	// update producer if necessary
	if kmprod.brokers != kmprod.topics.CurrentCluster.Brokers {
		kmprod.brokers = kmprod.topics.CurrentCluster.Brokers
		kmprod.prodCh = kmprod.plat.GetProducerCh(ctx, kmprod.brokers, wg)
		kmprod.slog = kmprod.slog.With().
			Str("Brokers", kmprod.brokers).
			Logger()
	}
}

func (kmprod *KafkaManagedProducer) producerDirective() *rkcypb.ProducerDirective {
	return &rkcypb.ProducerDirective{
		Id:          kmprod.id,
		ConcernName: kmprod.concernName,
		ConcernType: kmprod.concern.Concern.Type,
		Topic:       kmprod.topicName,
		Generation:  kmprod.topics.Topics.Current.Generation,
	}
}

func (kmprod *KafkaManagedProducer) Produce(
	directive rkcypb.Directive,
	traceParent string,
	key []byte,
	value []byte,
	deliveryChan chan kafka.Event,
) {
	kmprod.produceCh <- &message{
		directive:   directive,
		traceParent: traceParent,
		key:         key,
		value:       value,
		deliveryCh:  deliveryChan,
	}
}

func (kmprod *KafkaManagedProducer) Close() {
	kmprod.doneCh <- struct{}{}
}

func (kmprod *KafkaManagedProducer) run(ctx context.Context, wg *sync.WaitGroup) {
	pingAdminTicker := time.NewTicker(kmprod.plat.AdminPingInterval())
	pingMsg, err := rkcy.NewKafkaMessage(
		&kmprod.producersTopic,
		0,
		kmprod.producerDirective(),
		rkcypb.Directive_PRODUCER_STATUS,
		"",
	)
	if err != nil {
		kmprod.slog.Fatal().
			Err(err).
			Msg("Failed to create pingMsg")
	}

	paused := false
	for {
		if !paused {
			select {
			case <-ctx.Done():
				return
			case <-kmprod.doneCh:
				return
			case <-pingAdminTicker.C:
				kmprod.adminProdCh <- pingMsg
			case paused = <-kmprod.pauseCh:
				var directive rkcypb.Directive
				if paused {
					directive = rkcypb.Directive_PRODUCER_PAUSED
				} else {
					directive = rkcypb.Directive_PRODUCER_RUNNING
				}
				msg, err := rkcy.NewKafkaMessage(
					&kmprod.producersTopic,
					0,
					kmprod.producerDirective(),
					directive,
					"",
				)
				if err != nil {
					kmprod.slog.Error().
						Err(err).
						Msg("Failed to kafkaMessage")
					continue
				}
				log.Info().Msgf("pause produce %+v", msg)
				kmprod.adminProdCh <- msg
			case platformMsg := <-kmprod.platformCh:
				kmprod.updatePlatform(ctx, platformMsg.NewRtPlatDef, wg)
			case msg := <-kmprod.produceCh:
				if kmprod.prodCh == nil {
					kmprod.slog.Error().
						Msg("Failed to Produce, kafka Producer is nil")
					continue
				}

				kmprod.fnv64.Reset()
				kmprod.fnv64.Write(msg.key)
				fnvCalc := kmprod.fnv64.Sum64()
				partition := int32(fnvCalc % uint64(kmprod.topics.Topics.Current.PartitionCount))

				kMsg := &kafka.Message{
					TopicPartition: kafka.TopicPartition{
						Topic:     &kmprod.topics.CurrentTopic,
						Partition: partition,
					},
					Value:   msg.value,
					Headers: rkcy.StandardHeaders(msg.directive, msg.traceParent),
				}

				kmprod.prodCh <- kMsg
			}
		} else {
			// Same select as above without the produceCh read
			// This way we can hang in this bottom select until unpaused
			// and not read from produceCh, and producers will be paused
			// trying to publish to the channel as well.
			select {
			case <-ctx.Done():
				return
			case <-kmprod.doneCh:
				return
			case paused = <-kmprod.pauseCh:
				continue
			case platMsg := <-kmprod.platformCh:
				kmprod.updatePlatform(ctx, platMsg.NewRtPlatDef, wg)
			}
		}
	}
}
