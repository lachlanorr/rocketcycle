// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type FullMessage struct {
	Directive   Directive
	Value       []byte
	Offset      int64
	Timestamp   time.Time
	TraceParent string
}

type MessageHandler func(fullMsg *FullMessage)

type MatchLoc int

const (
	kPastLastMatch MatchLoc = 0
	kAtLastMatch            = 1
)

type FindResult int

const (
	kContinue FindResult = 0
	kFound               = 1
	kStop                = 2
)

func findMostRecentMatching(
	bootstrapServers string,
	topic string,
	partition int32,
	match Directive,
	matchLoc MatchLoc,
	delta int64,
) (FindResult, int64, error) {
	groupName := uncommittedGroupName(topic, int(partition))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kafkaLogCh := make(chan kafka.LogEvent)
	go printKafkaLogs(ctx, kafkaLogCh)

	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        bootstrapServers,
		"group.id":                 groupName,
		"enable.auto.commit":       false,
		"enable.auto.offset.store": false,

		"go.logs.channel.enable": true,
		"go.logs.channel":        kafkaLogCh,
	})
	if err != nil {
		return kStop, 0, err
	}
	defer func() {
		go cons.Close()
	}()

	low, high, err := cons.QueryWatermarkOffsets(topic, 0, 10000)
	if err != nil {
		return kStop, 0, err
	}

	if matchLoc == kPastLastMatch {
		return kFound, high, nil
	}

	if match == Directive_ALL {
		matchingOffset := high
		if matchLoc == kAtLastMatch {
			matchingOffset = maxi64(0, matchingOffset-1)
		}
		return kFound, matchingOffset, nil
	}

	startOffset := maxi64(low, high-delta)
	err = cons.Assign([]kafka.TopicPartition{
		{
			Topic:     &topic,
			Partition: partition,
			Offset:    kafka.Offset(startOffset),
		},
	})

	lastRead := startOffset
	matchingOffset := int64(-1)
	for {
		msg, err := cons.ReadMessage(5 * time.Second)
		timedOut := err != nil && err.(kafka.Error).Code() == kafka.ErrTimedOut
		if timedOut {
			return kStop, 0, err
		}
		if err != nil && !timedOut {
			log.Warn().Msgf("ReadMessage error %s", err.Error())
			return kStop, 0, err
		}
		if !timedOut && msg != nil {
			lastRead = int64(msg.TopicPartition.Offset)
			directive := GetDirective(msg)
			if (directive & match) == match {
				matchingOffset = int64(msg.TopicPartition.Offset)
			}
			if lastRead >= high-1 {
				break
			}
		}
	}

	if matchingOffset != -1 {
		if matchLoc == kPastLastMatch {
			matchingOffset++
		}
		return kFound, matchingOffset, nil
	} else {
		// if we didn't find it, return high
		if delta < high {
			return kContinue, high, nil
		} else {
			return kStop, high, nil
		}
	}
}

func FindMostRecentMatching(
	bootstrapServers string,
	topic string,
	partition int32,
	match Directive,
	matchLoc MatchLoc,
) (bool, int64, error) {
	const maxDelta int64 = 100000
	delta := int64(1)

	var (
		found FindResult = kContinue
		mro   int64
		err   error
	)

	for delta < maxDelta {
		found, mro, err = findMostRecentMatching(
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

		if found != kContinue {
			break
		} else {
			delta *= 10
			log.Warn().
				Str("Topic", topic).
				Int("Partition", int(partition)).
				Str("Match", match.String()).
				Msgf("FindMostRecentMatching Not found, new delta: %d", delta)
		}
	}

	log.Info().
		Str("Topic", topic).
		Int("Partition", int(partition)).
		Str("Match", match.String()).
		Msgf("FindMostRecentMatching, found: %t, offset: %d", found == kFound, mro)
	return found == kFound, mro, nil
}

func consumeAdminTopic(
	ctx context.Context,
	adminBrokers string,
	topic string,
	match Directive,
	startMatchLoc MatchLoc,
	handler MessageHandler,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	groupName := uncommittedGroupName(topic, 0)

	slog := log.With().
		Str("Topic", topic).
		Str("GroupName", groupName).
		Str("Match", match.String()).
		Logger()

	found, lastMatchOff, err := FindMostRecentMatching(
		adminBrokers,
		topic,
		0,
		match,
		startMatchLoc,
	)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("FindMostRecentOffset error")
	}
	if !found {
		slog.Fatal().
			Msg("No matching found with FindMostRecentOffset")
	}

	kafkaLogCh := make(chan kafka.LogEvent)
	go printKafkaLogs(ctx, kafkaLogCh)
	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        adminBrokers,
		"group.id":                 groupName,
		"enable.auto.commit":       false,
		"enable.auto.offset.store": false,

		"go.logs.channel.enable": true,
		"go.logs.channel":        kafkaLogCh,
	})
	if err != nil {
		slog.Error().
			Err(err).
			Msg("Failed to NewConsumer")
		return
	}
	defer func() {
		slog.Warn().
			Msgf("Closing kafka consumer")
		cons.Close()
		slog.Warn().
			Msgf("Closed kafka consumer")
	}()

	err = cons.Assign([]kafka.TopicPartition{
		{
			Topic:     &topic,
			Partition: 0,
			Offset:    kafka.Offset(lastMatchOff),
		},
	})

	if err != nil {
		slog.Error().
			Err(err).
			Msg("Failed to Assign")
		return
	}

	for {
		select {
		case <-ctx.Done():
			slog.Warn().
				Msg("consumeAdminTopic exiting, ctx.Done()")
			return
		default:
			msg, err := cons.ReadMessage(100 * time.Millisecond)
			timedOut := err != nil && err.(kafka.Error).Code() == kafka.ErrTimedOut
			if err != nil && !timedOut {
				slog.Error().
					Err(err).
					Msg("Error during ReadMessage")
			} else if !timedOut && msg != nil {
				directive := GetDirective(msg)
				if (directive & match) != 0 {
					fullMsg := &FullMessage{
						Directive:   directive,
						Value:       msg.Value,
						Offset:      int64(msg.TopicPartition.Offset),
						Timestamp:   msg.Timestamp,
						TraceParent: GetTraceParent(msg),
					}
					handler(fullMsg)
				}
			}
		}
	}
}

type PlatformMessage struct {
	Directive Directive
	Timestamp time.Time
	Offset    int64
	NewRtPlat *rtPlatform
	OldRtPlat *rtPlatform
}

func consumePlatformTopic(
	ctx context.Context,
	ch chan<- *PlatformMessage,
	adminBrokers string,
	platformName string,
	environment string,
	wg *sync.WaitGroup,
) {
	var currRtPlat *rtPlatform

	wg.Add(1)
	go consumeAdminTopic(
		ctx,
		adminBrokers,
		PlatformTopic(platformName, environment),
		Directive_PLATFORM,
		kAtLastMatch,
		func(fullMsg *FullMessage) {
			plat := &Platform{}
			err := proto.Unmarshal(fullMsg.Value, plat)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to Unmarshal Platform")
				return
			}

			rtPlat, err := newRtPlatform(plat)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to newRtPlatform")
				return
			}

			if currRtPlat != nil {
				if rtPlat.Hash == currRtPlat.Hash {
					// this happens frequently when admin replublishes
					return
				}
				if !rtPlat.Platform.UpdateTime.AsTime().After(currRtPlat.Platform.UpdateTime.AsTime()) {
					log.Info().
						Msgf("Platform not newer: old(%s) vs new(%s)", currRtPlat.Platform.UpdateTime.AsTime(), rtPlat.Platform.UpdateTime.AsTime())
					return
				}
			}

			ch <- &PlatformMessage{
				Directive: fullMsg.Directive,
				Timestamp: fullMsg.Timestamp,
				Offset:    fullMsg.Offset,
				NewRtPlat: rtPlat,
				OldRtPlat: currRtPlat,
			}
		},
		wg,
	)
}

type ConfigPublishMessage struct {
	Directive Directive
	Timestamp time.Time
	Offset    int64
	Config    *Config
}

func consumeConfigTopic(
	ctx context.Context,
	chPublish chan<- *ConfigPublishMessage,
	adminBrokers string,
	platformName string,
	environment string,
	wg *sync.WaitGroup,
) {
	wg.Add(1)
	go consumeAdminTopic(
		ctx,
		adminBrokers,
		ConfigTopic(platformName, environment),
		Directive_CONFIG,
		kAtLastMatch,
		func(fullMsg *FullMessage) {
			if chPublish != nil && (fullMsg.Directive&Directive_CONFIG_PUBLISH) == Directive_CONFIG_PUBLISH {
				conf := &Config{}
				err := proto.Unmarshal(fullMsg.Value, conf)
				if err != nil {
					log.Error().
						Err(err).
						Msg("Failed to Unmarshal Config")
					return
				}

				chPublish <- &ConfigPublishMessage{
					Directive: fullMsg.Directive,
					Timestamp: fullMsg.Timestamp,
					Offset:    fullMsg.Offset,
					Config:    conf,
				}
			}
		},
		wg,
	)
}

type ConsumerMessage struct {
	Directive         Directive
	Timestamp         time.Time
	Offset            int64
	ConsumerDirective *ConsumerDirective
}

func consumeConsumersTopic(
	ctx context.Context,
	ch chan<- *ConsumerMessage,
	adminBrokers string,
	platformName string,
	environment string,
	wg *sync.WaitGroup,
) {
	wg.Add(1)
	go consumeAdminTopic(
		ctx,
		adminBrokers,
		ConsumersTopic(platformName, environment),
		Directive_CONSUMER,
		kPastLastMatch,
		func(fullMsg *FullMessage) {
			consDir := &ConsumerDirective{}
			err := proto.Unmarshal(fullMsg.Value, consDir)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to Unmarshal ConsumerDirective")
				return
			}

			ch <- &ConsumerMessage{
				Directive:         fullMsg.Directive,
				Timestamp:         fullMsg.Timestamp,
				Offset:            fullMsg.Offset,
				ConsumerDirective: consDir,
			}
		},
		wg,
	)
}

type ProducerMessage struct {
	Directive         Directive
	ProducerDirective *ProducerDirective
	Timestamp         time.Time
	Offset            int64
}

func consumeProducersTopic(
	ctx context.Context,
	ch chan<- *ProducerMessage,
	adminBrokers string,
	platformName string,
	environment string,
	wg *sync.WaitGroup,
) {
	wg.Add(1)
	go consumeAdminTopic(
		ctx,
		adminBrokers,
		ProducersTopic(platformName, environment),
		Directive_PRODUCER,
		kPastLastMatch,
		func(fullMsg *FullMessage) {
			prodDir := &ProducerDirective{}
			err := proto.Unmarshal(fullMsg.Value, prodDir)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to Unmarshal ProducerDirective")
				return
			}

			ch <- &ProducerMessage{
				Directive:         fullMsg.Directive,
				Timestamp:         fullMsg.Timestamp,
				Offset:            fullMsg.Offset,
				ProducerDirective: prodDir,
			}
		},
		wg,
	)
}
