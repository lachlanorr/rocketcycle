// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
)

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

	low, high, err := cons.QueryWatermarkOffsets(topic, 0, 10000)
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