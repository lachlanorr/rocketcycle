// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package consumer

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type RawMessage struct {
	Directive   rkcypb.Directive
	Value       []byte
	Offset      int64
	Timestamp   time.Time
	TraceParent string
}

type MatchLoc int

const (
	PAST_LAST_MATCH MatchLoc = 0
	AT_LAST_MATCH            = 1
)

type FindResult int

const (
	CONTINUE FindResult = 0
	FOUND               = 1
	STOP                = 2
)

func findMostRecentMatching(
	plat rkcy.Platform,
	bootstrapServers string,
	topic string,
	partition int32,
	match rkcypb.Directive,
	matchLoc MatchLoc,
	delta int64,
) (FindResult, int64, error) {
	groupName := rkcy.UncommittedGroupName(topic, int(partition))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kafkaLogCh := make(chan kafka.LogEvent)
	go rkcy.PrintKafkaLogs(ctx, kafkaLogCh)

	cons, err := plat.NewConsumer(bootstrapServers, groupName, kafkaLogCh)
	if err != nil {
		return STOP, 0, err
	}
	defer func() {
		go cons.Close()
	}()

	low, high, err := cons.QueryWatermarkOffsets(topic, 0, 10000)
	if err != nil {
		return STOP, 0, err
	}

	if matchLoc == PAST_LAST_MATCH {
		return FOUND, high, nil
	}

	if match == rkcypb.Directive_ALL {
		matchingOffset := high
		if matchLoc == AT_LAST_MATCH {
			matchingOffset = rkcy.Maxi64(0, matchingOffset-1)
		}
		return FOUND, matchingOffset, nil
	}

	startOffset := rkcy.Maxi64(low, high-delta)
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
			return STOP, 0, err
		}
		if err != nil && !timedOut {
			log.Warn().Msgf("ReadMessage error %s", err.Error())
			return STOP, 0, err
		}
		if !timedOut && msg != nil {
			lastRead = int64(msg.TopicPartition.Offset)
			directive := rkcy.GetDirective(msg)
			if (directive & match) == match {
				matchingOffset = int64(msg.TopicPartition.Offset)
			}
			if lastRead >= high-1 {
				break
			}
		}
	}

	if matchingOffset != -1 {
		if matchLoc == PAST_LAST_MATCH {
			matchingOffset++
		}
		return FOUND, matchingOffset, nil
	} else {
		// if we didn't find it, return high
		if delta < high {
			return CONTINUE, high, nil
		} else {
			return STOP, high, nil
		}
	}
}

func FindMostRecentMatching(
	plat rkcy.Platform,
	bootstrapServers string,
	topic string,
	partition int32,
	match rkcypb.Directive,
	matchLoc MatchLoc,
) (bool, int64, error) {
	const MAX_DELTA int64 = 100000
	delta := int64(1)

	var (
		found FindResult = CONTINUE
		mro   int64
		err   error
	)

	for delta < MAX_DELTA {
		found, mro, err = findMostRecentMatching(
			plat,
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

		if found != CONTINUE {
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
		Msgf("FindMostRecentMatching, found: %t, offset: %d", found == FOUND, mro)
	return found == FOUND, mro, nil
}

// consumeMgmtTopic is intended for single paritition topics used in
// the management of the system. Examples include platform, consumers,
// producers, and config topics.
func ConsumeMgmtTopic(
	ctx context.Context,
	plat rkcy.Platform,
	topic string,
	match rkcypb.Directive,
	startMatchLoc MatchLoc,
	handler func(rawMsg *RawMessage),
	readyCh chan<- bool,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	groupName := rkcy.UncommittedGroupName(topic, 0)

	slog := log.With().
		Str("Topic", topic).
		Str("GroupName", groupName).
		Str("Match", match.String()).
		Logger()

	found, lastMatchOff, err := FindMostRecentMatching(
		plat,
		plat.AdminBrokers(),
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
	go rkcy.PrintKafkaLogs(ctx, kafkaLogCh)
	cons, err := plat.NewConsumer(plat.AdminBrokers(), groupName, kafkaLogCh)
	if err != nil {
		slog.Error().
			Err(err).
			Msg("Failed to NewConsumer")
		return
	}
	defer func() {
		slog.Warn().
			Str("Topic", topic).
			Msgf("CONSUMER Closing...")
		err := cons.Close()
		if err != nil {
			log.Error().
				Err(err).
				Str("Topic", topic).
				Msgf("Error during consumer.Close()")
		}
		slog.Warn().
			Str("Topic", topic).
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
				directive := rkcy.GetDirective(msg)
				if (directive & match) != 0 {
					rawMsg := &RawMessage{
						Directive:   directive,
						Value:       msg.Value,
						Offset:      int64(msg.TopicPartition.Offset),
						Timestamp:   msg.Timestamp,
						TraceParent: rkcy.GetTraceParent(msg),
					}
					handler(rawMsg)
				}
			}
		}
	}
}

// consumeACETopic behaves much like consumeMgmtTopic, but is intended
// to operate upon Admin, Complete, and Error topics belonging to
// concerns which are always single partition but may contain
// divergent generational definitions. The platform topic is read as
// well, and if the ACE topic being consumed changes definitions,
// consumeACETopic will automatically adjust to the new current topic
// definition.
func ConsumeACETopic(
	ctx context.Context,
	plat rkcy.Platform,
	concern string,
	aceTopic rkcy.StandardTopicName,

	match rkcypb.Directive,
	startMatchLoc MatchLoc,
	handler func(rawMsg *RawMessage),

	readyCh chan<- bool,
	wg *sync.WaitGroup,
) {
	if !rkcy.IsACETopic(string(aceTopic)) {
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
	platCh := make(chan *rkcy.PlatformMessage)
	ConsumePlatformTopic(
		ctx,
		plat,
		platCh,
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
			if (platMsg.Directive & rkcypb.Directive_PLATFORM) != rkcypb.Directive_PLATFORM {
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
				go ConsumeMgmtTopic(
					ctxACE,
					plat,
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
