// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"hash"
	"hash/fnv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type KafkaProducer struct {
	rawProducer *RawProducer

	id string

	constlog zerolog.Logger
	slog     zerolog.Logger

	platformName string
	concernName  string
	concern      *rtConcern
	topicName    string

	adminPingInterval time.Duration

	brokers string
	prodCh  ProducerCh
	topics  *rtTopics

	producersTopic string
	adminProdCh    ProducerCh

	doneCh     chan struct{}
	pauseCh    chan bool
	platformCh chan *PlatformMessage
	produceCh  chan *message

	fnv64 hash.Hash64
}

type message struct {
	directive   Directive
	traceParent string
	key         []byte
	value       []byte
	deliveryCh  chan kafka.Event
}

func newKafkaMessage(
	topic *string,
	partition int32,
	value proto.Message,
	directive Directive,
	traceParent string,
) (*kafka.Message, error) {
	valueSer, err := proto.Marshal(value)
	if err != nil {
		return nil, err
	}
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: topic, Partition: partition},
		Value:          valueSer,
		Headers:        standardHeaders(directive, traceParent),
	}
	//	log.Info().Msgf("msg: %+v", *msg)
	return msg, nil
}

func NewKafkaProducer(
	ctx context.Context,
	rawProducer *RawProducer,
	adminBrokers string,
	platformName string,
	environment string,
	concernName string,
	topicName string,
	adminPingInterval time.Duration,
	wg *sync.WaitGroup,
) *KafkaProducer {
	prod := KafkaProducer{
		rawProducer: rawProducer,
		id:          uuid.NewString(),
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
	prod.platformCh = make(chan *PlatformMessage)
	prod.produceCh = make(chan *message)

	prod.producersTopic = ProducersTopic(platformName, environment)
	prod.adminProdCh = prod.rawProducer.getProducerCh(ctx, adminBrokers, wg)

	consumePlatformTopic(
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

func (kprod *KafkaProducer) updatePlatform(
	ctx context.Context,
	rtPlatDef *rtPlatformDef,
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
		kprod.prodCh = kprod.rawProducer.getProducerCh(ctx, kprod.brokers, wg)
		kprod.slog = kprod.slog.With().
			Str("Brokers", kprod.brokers).
			Logger()
	}
}

func (kprod *KafkaProducer) producerDirective() *ProducerDirective {
	return &ProducerDirective{
		Id:          kprod.id,
		ConcernName: kprod.concernName,
		ConcernType: kprod.concern.Concern.Type,
		Topic:       kprod.topicName,
		Generation:  kprod.topics.Topics.Current.Generation,
	}
}

func (kprod *KafkaProducer) Produce(
	directive Directive,
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
	pingAdminTicker := time.NewTicker(kprod.adminPingInterval)
	pingMsg, err := newKafkaMessage(
		&kprod.producersTopic,
		0,
		kprod.producerDirective(),
		Directive_PRODUCER_STATUS,
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
				var directive Directive
				if paused {
					directive = Directive_PRODUCER_PAUSED
				} else {
					directive = Directive_PRODUCER_RUNNING
				}
				msg, err := newKafkaMessage(
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
					Headers: standardHeaders(msg.directive, msg.traceParent),
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
