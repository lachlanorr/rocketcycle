// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package stream

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/mgmt"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type ManagedProducer struct {
	platArgs *rkcy.PlatformArgs
	bprod    *BrokersProducer

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

	pauseCh    chan bool
	platformCh chan *mgmt.PlatformMessage
	produceCh  chan *message

	hashFunc rkcy.HashFunc
}

type message struct {
	directive   rkcypb.Directive
	traceParent string
	key         []byte
	value       []byte
	deliveryCh  chan kafka.Event
}

func NewManagedProducer(
	ctx context.Context,
	wg *sync.WaitGroup,
	platArgs *rkcy.PlatformArgs,
	bprod *BrokersProducer,
	concernName string,
	topicName string,
) *ManagedProducer {
	mprod := ManagedProducer{
		platArgs: platArgs,
		bprod:    bprod,
		id:       uuid.NewString(),
		constlog: log.With().
			Str("Concern", concernName).
			Logger(),
		concernName: concernName,
		topicName:   topicName,
		hashFunc:    rkcy.GetHashFunc("fnv64"),
	}

	mprod.slog = mprod.constlog.With().Logger()
	mprod.pauseCh = make(chan bool)
	mprod.platformCh = make(chan *mgmt.PlatformMessage)
	mprod.produceCh = make(chan *message)

	mprod.producersTopic = rkcy.ProducersTopic(platArgs.Platform, platArgs.Environment)
	mprod.adminProdCh = mprod.bprod.GetProducerCh(ctx, wg, platArgs.AdminBrokers)

	mgmt.ConsumePlatformTopic(
		ctx,
		wg,
		bprod.StreamProvider,
		platArgs.Platform,
		platArgs.Environment,
		platArgs.AdminBrokers,
		mprod.platformCh,
		nil,
	)

	platMsg := <-mprod.platformCh
	mprod.updatePlatform(ctx, wg, platMsg.NewRtPlatDef)

	go mprod.run(ctx, wg)

	return &mprod
}

func (mprod *ManagedProducer) ConcernName() string {
	return mprod.concernName
}

func (mprod *ManagedProducer) Concern() *rkcy.RtConcern {
	return mprod.concern
}

func (mprod *ManagedProducer) TopicName() string {
	return mprod.topicName
}

func (mprod *ManagedProducer) Topics() *rkcy.RtTopics {
	return mprod.topics
}

func (mprod *ManagedProducer) HashFunc() rkcy.HashFunc {
	return mprod.hashFunc
}

func (mprod *ManagedProducer) updatePlatform(
	ctx context.Context,
	wg *sync.WaitGroup,
	rtPlatDef *rkcy.RtPlatformDef,
) {
	var ok bool
	mprod.concern, ok = rtPlatDef.Concerns[mprod.concernName]
	if !ok {
		mprod.slog.Error().
			Str("Concern", mprod.concernName).
			Msg("updatePlatform: Failed to find Concern")
		return
	}

	mprod.topics, ok = mprod.concern.Topics[mprod.topicName]
	if !ok {
		mprod.slog.Error().
			Str("Topic", mprod.topicName).
			Msg("updatePlatform: Failed to find Topics")
		return
	}

	// reset to copy of constlog since we will replace "Topic" sometimes
	mprod.slog = mprod.constlog.With().Str("Topic", mprod.topics.CurrentTopic).Logger()

	// update producer if necessary
	if mprod.brokers != mprod.topics.CurrentCluster.Brokers {
		mprod.brokers = mprod.topics.CurrentCluster.Brokers
		mprod.prodCh = mprod.bprod.GetProducerCh(ctx, wg, mprod.brokers)
		mprod.slog = mprod.slog.With().
			Str("Brokers", mprod.brokers).
			Logger()
	}
}

func (mprod *ManagedProducer) producerDirective() *rkcypb.ProducerDirective {
	return &rkcypb.ProducerDirective{
		Id:          mprod.id,
		ConcernName: mprod.concernName,
		ConcernType: mprod.concern.Concern.Type,
		Topic:       mprod.topicName,
		Generation:  mprod.topics.Topics.Current.Generation,
	}
}

func (mprod *ManagedProducer) Produce(
	directive rkcypb.Directive,
	traceParent string,
	key []byte,
	value []byte,
	deliveryChan chan kafka.Event,
) {
	mprod.produceCh <- &message{
		directive:   directive,
		traceParent: traceParent,
		key:         key,
		value:       value,
		deliveryCh:  deliveryChan,
	}
}

func prepKafkaMessage(
	mprod *ManagedProducer,
	directive rkcypb.Directive,
	traceParent string,
	key []byte,
	value []byte,
) *kafka.Message {
	partition := mprod.HashFunc().Hash(key, mprod.Topics().Topics.Current.PartitionCount)

	kMsg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &mprod.Topics().CurrentTopic,
			Partition: partition,
		},
		Value:   value,
		Headers: rkcy.StandardHeaders(directive, traceParent),
	}
	return kMsg
}

func (mprod *ManagedProducer) run(ctx context.Context, wg *sync.WaitGroup) {
	pingAdminTicker := time.NewTicker(mprod.platArgs.AdminPingInterval)
	pingMsg, err := rkcy.NewKafkaMessage(
		&mprod.producersTopic,
		0,
		mprod.producerDirective(),
		rkcypb.Directive_PRODUCER_STATUS,
		"",
	)
	if err != nil {
		mprod.slog.Fatal().
			Err(err).
			Msg("Failed to create pingMsg")
	}

	paused := false
	for {
		if !paused {
			select {
			case <-ctx.Done():
				return
			case <-pingAdminTicker.C:
				mprod.adminProdCh <- pingMsg
			case paused = <-mprod.pauseCh:
				var directive rkcypb.Directive
				if paused {
					directive = rkcypb.Directive_PRODUCER_PAUSED
				} else {
					directive = rkcypb.Directive_PRODUCER_RUNNING
				}
				msg, err := rkcy.NewKafkaMessage(
					&mprod.producersTopic,
					0,
					mprod.producerDirective(),
					directive,
					"",
				)
				if err != nil {
					mprod.slog.Error().
						Err(err).
						Msg("Failed to kafkaMessage")
					continue
				}
				log.Info().Msgf("pause produce %+v", msg)
				mprod.adminProdCh <- msg
			case platformMsg := <-mprod.platformCh:
				mprod.updatePlatform(ctx, wg, platformMsg.NewRtPlatDef)
			case msg := <-mprod.produceCh:
				if mprod.prodCh == nil {
					mprod.slog.Error().
						Msg("Failed to Produce, kafka Producer is nil")
					continue
				}

				kMsg := prepKafkaMessage(
					mprod,
					msg.directive,
					msg.traceParent,
					msg.key,
					msg.value,
				)

				mprod.prodCh <- kMsg
			}
		} else {
			// Same select as above without the produceCh read
			// This way we can hang in this bottom select until unpaused
			// and not read from produceCh, and producers will be paused
			// trying to publish to the channel as well.
			select {
			case <-ctx.Done():
				return
			case paused = <-mprod.pauseCh:
				continue
			case platMsg := <-mprod.platformCh:
				mprod.updatePlatform(ctx, wg, platMsg.NewRtPlatDef)
			}
		}
	}
}
