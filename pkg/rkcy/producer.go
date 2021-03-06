// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"hash"
	"hash/fnv"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type Producer struct {
	id string

	constlog zerolog.Logger
	slog     zerolog.Logger

	platformName string
	concernName  string
	concern      *rtConcern
	topicName    string

	brokers string
	prodCh  ProducerCh
	topics  *rtTopics

	adminTopic  string
	adminProdCh ProducerCh

	doneCh    chan struct{}
	pauseCh   chan bool
	adminCh   chan *AdminMessage
	produceCh chan *message

	fnv64 hash.Hash64
}

type message struct {
	directive   Directive
	traceParent string
	key         []byte
	value       []byte
	deliveryCh  chan kafka.Event
}

func kafkaMessage(
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

func NewProducer(
	ctx context.Context,
	adminBrokers string,
	platformName string,
	concernName string,
	topicName string,
) *Producer {
	prod := Producer{
		id: uuid.NewString(),
		constlog: log.With().
			Str("Concern", concernName).
			Logger(),
		platformName: platformName,
		concernName:  concernName,
		topicName:    topicName,
		fnv64:        fnv.New64(),
	}

	prod.slog = prod.constlog.With().Logger()
	prod.doneCh = make(chan struct{})
	prod.pauseCh = make(chan bool)
	prod.adminCh = make(chan *AdminMessage)
	prod.produceCh = make(chan *message)

	prod.adminTopic = AdminTopic(platformName)
	prod.adminProdCh = getProducerCh(adminBrokers)

	go consumeAdminTopic(ctx, prod.adminCh, adminBrokers, platformName, Directive_PLATFORM, Directive_PLATFORM, kAtLastMatch)

	adminMsg := <-prod.adminCh
	prod.updatePlatform(adminMsg.NewRtPlat)

	go prod.run(ctx)

	return &prod
}

func (prod *Producer) updatePlatform(rtPlat *rtPlatform) {
	var ok bool
	prod.concern, ok = rtPlat.Concerns[prod.concernName]
	if !ok {
		prod.slog.Error().
			Msg("updatePlatform: Failed to find Concern")
		return
	}

	prod.topics, ok = prod.concern.Topics[prod.topicName]
	if !ok {
		prod.slog.Error().
			Msg("updatePlatform: Failed to find Topics")
		return
	}

	// reset to copy of constlog since we will replace "Topic" sometimes
	prod.slog = prod.constlog.With().Str("Topic", prod.topics.CurrentTopic).Logger()

	// update producer if necessary
	if prod.brokers != prod.topics.CurrentCluster.Brokers {
		prod.brokers = prod.topics.CurrentCluster.Brokers
		prod.prodCh = getProducerCh(prod.brokers)
		prod.slog = prod.slog.With().
			Str("Brokers", prod.brokers).
			Logger()
	}
}

func (prod *Producer) producerDirective() *ProducerDirective {
	return &ProducerDirective{
		Id:          prod.id,
		ConcernName: prod.concernName,
		ConcernType: prod.concern.Concern.Type,
		Topic:       prod.topicName,
		Generation:  prod.topics.Topics.Current.Generation,
	}
}

func (prod *Producer) Produce(
	directive Directive,
	traceParent string,
	key []byte,
	value []byte,
	deliveryChan chan kafka.Event,
) {
	prod.produceCh <- &message{
		directive:   directive,
		traceParent: traceParent,
		key:         key,
		value:       value,
		deliveryCh:  deliveryChan,
	}
}

func (prod *Producer) Close() {
	prod.doneCh <- struct{}{}
}

func (prod *Producer) run(ctx context.Context) {
	pingAdminTicker := time.NewTicker(gAdminPingInterval)
	pingMsg, err := kafkaMessage(
		&prod.adminTopic,
		0,
		prod.producerDirective(),
		Directive_PRODUCER_STATUS,
		"",
	)
	if err != nil {
		prod.slog.Fatal().
			Err(err).
			Msg("Failed to create pingMsg")
	}

	paused := false
	for {
		if !paused {
			select {
			case <-ctx.Done():
				return
			case <-prod.doneCh:
				return
			case <-pingAdminTicker.C:
				prod.adminProdCh <- pingMsg
			case paused = <-prod.pauseCh:
				var directive Directive
				if paused {
					directive = Directive_PRODUCER_STOPPED
				} else {
					directive = Directive_PRODUCER_STARTED
				}
				msg, err := kafkaMessage(
					&prod.adminTopic,
					0,
					prod.producerDirective(),
					directive,
					"",
				)
				if err != nil {
					prod.slog.Error().
						Err(err).
						Msg("Failed to kafkaMessage")
					continue
				}
				log.Info().Msgf("pause produce %+v", msg)
				prod.adminProdCh <- msg
			case adminMsg := <-prod.adminCh:
				prod.updatePlatform(adminMsg.NewRtPlat)
			case msg := <-prod.produceCh:
				if prod.prodCh == nil {
					prod.slog.Error().
						Msg("Failed to Produce, kafka Producer is nil")
					continue
				}

				prod.fnv64.Reset()
				prod.fnv64.Write(msg.key)
				fnvCalc := prod.fnv64.Sum64()
				partition := int32(fnvCalc % uint64(prod.topics.Topics.Current.PartitionCount))

				kMsg := &kafka.Message{
					TopicPartition: kafka.TopicPartition{
						Topic:     &prod.topics.CurrentTopic,
						Partition: partition,
					},
					Value:   msg.value,
					Headers: standardHeaders(msg.directive, msg.traceParent),
				}

				prod.prodCh <- kMsg
			}
		} else {
			// Same select as above without the produceCh read
			// This way we can hang in this bottom select until unpaused
			// and not read from produceCh, and producers will be paused
			// trying to publish to the channel as well.
			select {
			case <-ctx.Done():
				return
			case <-prod.doneCh:
				return
			case paused = <-prod.pauseCh:
				continue
			case adminMsg := <-prod.adminCh:
				prod.updatePlatform(adminMsg.NewRtPlat)
			}
		}
	}
}
