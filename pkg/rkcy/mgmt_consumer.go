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

type RawMessage struct {
	Directive   Directive
	Value       []byte
	Offset      int64
	Timestamp   time.Time
	TraceParent string
}

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

	log.Debug().
		Str("Topic", topic).
		Int("Partition", int(partition)).
		Str("Match", match.String()).
		Msgf("FindMostRecentMatching, found: %t, offset: %d", found == kFound, mro)
	return found == kFound, mro, nil
}

// consumeMgmtTopic is intended for single paritition topics used in
// the management of the system. Examples include platform, consumers,
// producers, and config topics.
func consumeMgmtTopic(
	ctx context.Context,
	adminBrokers string,
	topic string,
	match Directive,
	startMatchLoc MatchLoc,
	handler func(rawMsg *RawMessage),
	readyCh chan<- bool,
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
			Msgf("CONSUMER Closing...")
		cons.Close()
		slog.Warn().
			Msgf("CONSUMER CLOSED")
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

	if readyCh != nil {
		readyCh <- true
	}

	for {
		select {
		case <-ctx.Done():
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
					rawMsg := &RawMessage{
						Directive:   directive,
						Value:       msg.Value,
						Offset:      int64(msg.TopicPartition.Offset),
						Timestamp:   msg.Timestamp,
						TraceParent: GetTraceParent(msg),
					}
					handler(rawMsg)
				}
			}
		}
	}
}

type PlatformMessage struct {
	Directive    Directive
	Timestamp    time.Time
	Offset       int64
	NewRtPlatDef *rtPlatformDef
	OldRtPlatDef *rtPlatformDef
}

func consumePlatformTopic(
	ctx context.Context,
	ch chan<- *PlatformMessage,
	adminBrokers string,
	platformName string,
	environment string,
	readyCh chan<- bool,
	wg *sync.WaitGroup,
) {
	var currRtPlatDef *rtPlatformDef

	wg.Add(1)
	go consumeMgmtTopic(
		ctx,
		adminBrokers,
		PlatformTopic(platformName, environment),
		Directive_PLATFORM,
		kAtLastMatch,
		func(rawMsg *RawMessage) {
			platDef := &PlatformDef{}
			err := proto.Unmarshal(rawMsg.Value, platDef)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to Unmarshal Platform")
				return
			}

			rtPlatDef, err := newRtPlatformDef(platDef, platformName, environment)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to newRtPlatform")
				return
			}

			if currRtPlatDef != nil {
				if rtPlatDef.Hash == currRtPlatDef.Hash {
					// this happens frequently when admin replublishes
					return
				}
				if !rtPlatDef.PlatformDef.UpdateTime.AsTime().After(currRtPlatDef.PlatformDef.UpdateTime.AsTime()) {
					log.Info().
						Msgf("Platform not newer: old(%s) vs new(%s)", currRtPlatDef.PlatformDef.UpdateTime.AsTime(), rtPlatDef.PlatformDef.UpdateTime.AsTime())
					return
				}
			}

			ch <- &PlatformMessage{
				Directive:    rawMsg.Directive,
				Timestamp:    rawMsg.Timestamp,
				Offset:       rawMsg.Offset,
				NewRtPlatDef: rtPlatDef,
				OldRtPlatDef: currRtPlatDef,
			}
		},
		readyCh,
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
	readyCh chan<- bool,
	wg *sync.WaitGroup,
) {
	wg.Add(1)
	go consumeMgmtTopic(
		ctx,
		adminBrokers,
		ConfigTopic(platformName, environment),
		Directive_CONFIG,
		kAtLastMatch,
		func(rawMsg *RawMessage) {
			if chPublish != nil && (rawMsg.Directive&Directive_CONFIG_PUBLISH) == Directive_CONFIG_PUBLISH {
				conf := &Config{}
				err := proto.Unmarshal(rawMsg.Value, conf)
				if err != nil {
					log.Error().
						Err(err).
						Msg("Failed to Unmarshal Config")
					return
				}

				chPublish <- &ConfigPublishMessage{
					Directive: rawMsg.Directive,
					Timestamp: rawMsg.Timestamp,
					Offset:    rawMsg.Offset,
					Config:    conf,
				}
			}
		},
		readyCh,
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
	readyCh chan<- bool,
	wg *sync.WaitGroup,
) {
	wg.Add(1)
	go consumeMgmtTopic(
		ctx,
		adminBrokers,
		ConsumersTopic(platformName, environment),
		Directive_CONSUMER,
		kPastLastMatch,
		func(rawMsg *RawMessage) {
			consDir := &ConsumerDirective{}
			err := proto.Unmarshal(rawMsg.Value, consDir)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to Unmarshal ConsumerDirective")
				return
			}

			ch <- &ConsumerMessage{
				Directive:         rawMsg.Directive,
				Timestamp:         rawMsg.Timestamp,
				Offset:            rawMsg.Offset,
				ConsumerDirective: consDir,
			}
		},
		readyCh,
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
	readyCh chan<- bool,
	wg *sync.WaitGroup,
) {
	wg.Add(1)
	go consumeMgmtTopic(
		ctx,
		adminBrokers,
		ProducersTopic(platformName, environment),
		Directive_PRODUCER,
		kPastLastMatch,
		func(rawMsg *RawMessage) {
			prodDir := &ProducerDirective{}
			err := proto.Unmarshal(rawMsg.Value, prodDir)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to Unmarshal ProducerDirective")
				return
			}

			ch <- &ProducerMessage{
				Directive:         rawMsg.Directive,
				Timestamp:         rawMsg.Timestamp,
				Offset:            rawMsg.Offset,
				ProducerDirective: prodDir,
			}
		},
		readyCh,
		wg,
	)
}

// consumeACETopic behaves much like consumeMgmtTopic, but is intended
// to operate upon Admin, Complete, and Error topics belonging to
// concerns which are always single partition but may contain
// divergent generational definitions. The platform topic is read as
// well, and if the ACE topic being consumed changes definitions,
// consumeACETopic will automatically adjust to the new current topic
// definition.
func consumeACETopic(
	ctx context.Context,
	adminBrokers string,
	platformName string,
	environment string,
	concern string,
	aceTopic StandardTopicName,

	match Directive,
	startMatchLoc MatchLoc,
	handler func(rawMsg *RawMessage),

	readyCh chan<- bool,
	wg *sync.WaitGroup,
) {
	if !isACETopic(string(aceTopic)) {
		log.Fatal().Msgf("consumeConcernACETopic invalid topic: %s", aceTopic)
	}

	var (
		topicACE  string
		ctxACE    context.Context
		cancelACE context.CancelFunc
		wgACE     *sync.WaitGroup
	)

	// consume platform topic so we can read messages off the correct
	// concern admin physical topic
	platCh := make(chan *PlatformMessage)
	consumePlatformTopic(
		ctx,
		platCh,
		adminBrokers,
		platformName,
		environment,
		nil,
		wg,
	)

	for {
		select {
		case <-ctx.Done():
			if cancelACE != nil {
				cancelACE()
				wgACE.Wait()
			}
			return
		case platMsg := <-platCh:
			if (platMsg.Directive & Directive_PLATFORM) != Directive_PLATFORM {
				log.Error().Msgf("Invalid directive for PlatformTopic: %s", platMsg.Directive.String())
				continue
			}

			rtPlatDef := platMsg.NewRtPlatDef
			rtCnc, ok := rtPlatDef.Concerns[concern]
			if !ok {
				log.Error().Msgf("Concern not found in platform: %s", concern)
				continue
			}

			rtTop, ok := rtCnc.Topics[string(aceTopic)]
			if !ok {
				log.Error().Msgf("ACE topic '%s' not found in concern: %s", aceTopic, concern)
				continue
			}

			if rtTop.CurrentTopicPartitionCount != 1 {
				log.Error().Msgf("ACE topic '%s' invalid partition count in concern: %s", aceTopic, concern)
			}

			if rtTop.CurrentTopic != topicACE {
				if cancelACE != nil {
					cancelACE()
					wgACE.Wait()
				}
				topicACE = rtTop.CurrentTopic
				ctxACE, cancelACE = context.WithCancel(ctx)
				wgACE = &sync.WaitGroup{}
				wgACE.Add(1)
				go consumeMgmtTopic(
					ctxACE,
					adminBrokers,
					topicACE,
					match,
					startMatchLoc,
					handler,
					readyCh,
					wgACE,
				)
				readyCh = nil // only send ready on first consumeMgmtTopic call
			}
		}
	}
}

type ConcernAdminMessage struct {
	Directive             Directive
	Timestamp             time.Time
	Offset                int64
	ConcernAdminDirective *ConcernAdminDirective
}

func consumeConcernAdminTopic(
	ctx context.Context,
	ch chan<- *ConcernAdminMessage,
	adminBrokers string,
	platformName string,
	environment string,
	concern string,
	readyCh chan<- bool,
	wg *sync.WaitGroup,
) {
	go consumeACETopic(
		ctx,
		adminBrokers,
		platformName,
		environment,
		concern,
		ADMIN,
		Directive_CONCERN_ADMIN,
		kPastLastMatch,
		func(rawMsg *RawMessage) {
			cncAdminDir := &ConcernAdminDirective{}
			err := proto.Unmarshal(rawMsg.Value, cncAdminDir)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to Unmarshal ConcernAdminDirective")
				return
			}

			ch <- &ConcernAdminMessage{
				Directive:             rawMsg.Directive,
				Timestamp:             rawMsg.Timestamp,
				Offset:                rawMsg.Offset,
				ConcernAdminDirective: cncAdminDir,
			}
		},
		readyCh,
		wg,
	)
}
