// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
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

type MatchLoc int

const (
	PastLastMatch MatchLoc = 0
	AtLastMatch   MatchLoc = 1
)

func findMostRecentMatching(
	bootstrapServers string,
	topic string,
	partition int32,
	match pb.Directive,
	matchLoc MatchLoc,
	delta int64,
) (bool, int64, error) {
	groupName := uncommittedGroupName(topic, int(partition))
	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  bootstrapServers,
		"group.id":           groupName,
		"enable.auto.commit": false,
	})
	if err != nil {
		return false, 0, err
	}
	defer cons.Close()

	low, high, err := cons.QueryWatermarkOffsets(topic, 0, 5000)
	if err != nil {
		log.Error().
			Err(err).
			Str("Topic", topic).
			Msg("findMostRecentMatching: QueryWatermarkOffsets failed, topic likely doesn't exist yet, return 0 offset")
		return true, 0, nil
	}

	if matchLoc == PastLastMatch {
		return true, high, nil
	}

	if match == pb.Directive_ALL {
		matchingOffset := high
		if matchLoc == AtLastMatch {
			matchingOffset = maxi64(0, matchingOffset-1)
		}
		return true, matchingOffset, nil
	}

	err = cons.Assign([]kafka.TopicPartition{
		{
			Topic:     &topic,
			Partition: partition,
			Offset:    kafka.Offset(maxi64(low, high-delta)),
		},
	})

	lastRead := int64(0)
	matchingOffset := int64(-1)
	for lastRead < high-1 {
		msg, err := cons.ReadMessage(time.Second * 5)
		timedOut := err != nil && err.(kafka.Error).Code() == kafka.ErrTimedOut
		if timedOut {
			return false, 0, errors.New("findMostRecentMatching timed out")
		}
		if err != nil && !timedOut {
			return false, 0, err
		}
		if !timedOut && msg != nil {
			lastRead = int64(msg.TopicPartition.Offset)
			directive := getDirective(msg)
			if (directive & match) == match {
				matchingOffset = int64(msg.TopicPartition.Offset)
			}
		}
	}

	if matchingOffset != -1 {
		if matchLoc == PastLastMatch {
			matchingOffset++
		}
		return true, matchingOffset, nil
	} else {
		// if we didn't find it, return high
		return false, high, nil
	}
}

func FindMostRecentMatching(
	bootstrapServers string,
	topic string,
	partition int32,
	match pb.Directive,
	matchLoc MatchLoc,
) (bool, int64, error) {
	const maxDelta int64 = 100000
	delta := int64(100)

	var (
		found bool
		mro   int64
	)

	for delta < maxDelta {
		found, mro, err := findMostRecentMatching(
			bootstrapServers,
			topic,
			partition,
			match,
			matchLoc,
			delta,
		)
		if err != nil {
			return false, 0, err
		}
		if found {
			return found, mro, nil
		}
		delta *= 10
	}

	return found, mro, nil
}
