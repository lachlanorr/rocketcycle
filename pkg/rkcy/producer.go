// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"hash"
	"hash/fnv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
)

type Producer struct {
	constlog zerolog.Logger
	slog     zerolog.Logger

	platformName string
	concernName  string
	topicName    string

	clusterBrokers string
	kProd          *kafka.Producer
	topics         *rtTopics

	doneCh     chan struct{}
	platformCh chan *pb.Platform
	produceCh  chan *message

	fnv64 hash.Hash64
}

type message struct {
	directive  pb.Directive
	reqId      string
	key        []byte
	value      []byte
	deliveryCh chan kafka.Event
}

func NewProducer(
	ctx context.Context,
	bootstrapServers string,
	platformName string,
	concernName string,
	topicName string,
) *Producer {

	prod := Producer{
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
	prod.platformCh = make(chan *pb.Platform)
	prod.produceCh = make(chan *message)

	go consumePlatformConfig(ctx, prod.platformCh, bootstrapServers, platformName)

	plat := <-prod.platformCh
	prod.updatePlatform(plat)

	go prod.run(ctx)

	return &prod
}

func (prod *Producer) updatePlatform(plat *pb.Platform) {
	rtPlat, err := newRtPlatform(plat)
	if err != nil {
		prod.slog.Error().
			Err(err).
			Msg("updatePlatform: Failed to newRtPlatform")
		return
	}

	concern, ok := rtPlat.Concerns[prod.concernName]
	if !ok {
		prod.slog.Error().
			Msg("updatePlatform: Failed to find Concern")
		return
	}

	prod.topics, ok = concern.Topics[prod.topicName]
	if !ok {
		prod.slog.Error().
			Msg("updatePlatform: Failed to find Topics")
		return
	}

	// reset to copy of constlog since we will replace "Topic" sometimes
	prod.slog = prod.constlog.With().Str("Topic", prod.topics.CurrentTopic).Logger()

	// update producer if necessary
	if prod.clusterBrokers != prod.topics.CurrentCluster.BootstrapServers {
		prod.closeKProd()
		prod.clusterBrokers = prod.topics.CurrentCluster.BootstrapServers
		prod.slog = prod.slog.With().
			Str("ClusterBrokers", prod.clusterBrokers).
			Logger()
		prod.kProd, err = kafka.NewProducer(&kafka.ConfigMap{
			"bootstrap.servers": prod.clusterBrokers,
		})
		if err != nil {
			prod.kProd = nil
			prod.slog.Error().
				Err(err).
				Msg("failed to kafka.NewProducer")
			return
		}
	}
}

func (prod *Producer) Produce(
	directive pb.Directive,
	reqId string,
	key []byte,
	value []byte,
	deliveryChan chan kafka.Event,
) {
	prod.produceCh <- &message{
		directive:  directive,
		reqId:      reqId,
		key:        key,
		value:      value,
		deliveryCh: deliveryChan,
	}
}

func (prod *Producer) closeKProd() {
	if prod.kProd != nil {
		prod.kProd.Flush(5 * 1000)
		prod.kProd.Close()
	}
}

func (prod *Producer) Close() {
	prod.doneCh <- struct{}{}
}

func (prod *Producer) run(ctx context.Context) {
	defer prod.closeKProd()

	for {
		select {
		case <-ctx.Done():
			prod.slog.Info().
				Msg("Producer.run: exiting, ctx.Done()")
			return
		case <-prod.doneCh:
			prod.slog.Info().
				Msg("Producer.run: exiting, ctx.Done()")
			return
		case plat := <-prod.platformCh:
			prod.updatePlatform(plat)
		case msg := <-prod.produceCh:
			if prod.kProd == nil {
				prod.slog.Error().
					Msg("Failed to Produce, kafka Producer is nil")
				continue
			}

			prod.fnv64.Reset()
			prod.fnv64.Write(msg.key)
			fnvCalc := prod.fnv64.Sum64()
			partition := int32(fnvCalc % uint64(prod.topics.Topics.Current.PartitionCount))

			kMsg := kafka.Message{
				TopicPartition: kafka.TopicPartition{
					Topic:     &prod.topics.CurrentTopic,
					Partition: partition,
				},
				Value:   msg.value,
				Headers: standardHeaders(msg.directive, msg.reqId),
			}

			err := prod.kProd.Produce(&kMsg, msg.deliveryCh)
			if err != nil {
				prod.slog.Error().
					Err(err).
					Msg("Failed to Produce")
				continue
			}
		}
	}
}
