// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rkcy/pkg/rkcy/pb"
)

type Consumer struct {
	constlog zerolog.Logger
	slog     zerolog.Logger

	platformName string
	concernName  string
	topicName    string
	partition    int32

	clusterBrokers string
	kCons          *kafka.Consumer
	kTopic         string
	topics         *rtTopics

	platformCh chan *pb.Platform
}

func NewConsumer(
	ctx context.Context,
	bootstrapServers string,
	platformName string,
	concernName string,
	topicName string,
	partition int32,
) *Consumer {

	cons := Consumer{
		constlog: log.With().
			Str("Concern", concernName).
			Logger(),
		platformName: platformName,
		concernName:  concernName,
		topicName:    topicName,
		partition:    partition,
	}

	cons.slog = cons.constlog.With().Logger()

	go consumePlatformConfig(ctx, cons.platformCh, bootstrapServers, platformName)

	plat := <-cons.platformCh
	cons.updatePlatform(plat)

	return &cons
}

func (cons *Consumer) updatePlatform(plat *pb.Platform) {
	rtPlat, err := newRtPlatform(plat)
	if err != nil {
		cons.slog.Error().
			Err(err).
			Msg("updatePlatform: Failed to newRtPlatform")
		return
	}

	concern, ok := rtPlat.Concerns[cons.concernName]
	if !ok {
		cons.slog.Error().
			Msg("updatePlatform: Failed to find Concern")
		return
	}

	cons.topics, ok = concern.Topics[cons.topicName]
	if !ok {
		cons.slog.Error().
			Msg("updatePlatform: Failed to find Topics")
		return
	}
	if cons.partition >= cons.topics.Topics.Current.PartitionCount {
		cons.slog.Error().
			Msgf("updatePlatform: Invalid partition %d requested, count = %d", cons.partition, cons.topics.Topics.Current.PartitionCount)
		return
	}

	// reset to copy of constlog since we will replace "Topic" sometimes
	cons.slog = cons.constlog.With().Str("Topic", cons.topics.CurrentTopic).Logger()

	// update consumer if necessary
	if cons.kTopic != cons.topics.CurrentTopic ||
		cons.clusterBrokers != cons.topics.CurrentCluster.BootstrapServers {

		cons.kTopic = cons.topics.CurrentTopic

		cons.Close()
		cons.clusterBrokers = cons.topics.CurrentCluster.BootstrapServers
		cons.slog = cons.slog.With().
			Str("ClusterBrokers", cons.clusterBrokers).
			Logger()

		groupName := fmt.Sprintf("rkcy_%s", cons.kTopic)

		cons.kCons, err = kafka.NewConsumer(&kafka.ConfigMap{
			"bootstrap.servers":        cons.clusterBrokers,
			"group.id":                 groupName,
			"enable.auto.commit":       true,  // librdkafka will commit to brokers for us on an interval and when we close consumer
			"enable.auto.offset.store": false, // we explicitely commit to local store to get "at least once" behavior
		})
		if err != nil {
			cons.kCons = nil
			cons.slog.Error().
				Err(err).
				Msg("failed to kafka.NewConsumer")
			return
		}

		err = cons.kCons.Assign([]kafka.TopicPartition{
			{
				Topic:     &cons.kTopic,
				Partition: cons.partition,
			},
		})

		if err != nil {
			cons.slog.Error().
				Err(err).
				Msg("Failed to Assign")
			return
		}

	}
}

func (cons *Consumer) Close() {
	if cons.kCons != nil {
		cons.kCons.Close()
	}
}

func (cons *Consumer) ReadMessage(timeout time.Duration) (*kafka.Message, error) {
	return cons.kCons.ReadMessage(timeout)
}
