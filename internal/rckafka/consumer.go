// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rckafka

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	admin_pb "github.com/lachlanorr/rocketcycle/build/proto/admin"
)

func PlatformTopic(name string) string {
	return fmt.Sprintf("rc.%s.platform", name)
}

func ConsumePlatformConfig(ctx context.Context, ch chan<- admin_pb.Platform, bootstrapServers string, platformName string) {
	platformTopic := PlatformTopic(platformName)
	groupName := "__" + platformTopic + "__non_committed_group"

	slog := log.With().
		Str("bootstrap.servers", bootstrapServers).
		Str("Topic", platformTopic).
		Logger()

	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  "localhost",
		"group.id":           groupName,
		"enable.auto.commit": false,
	})
	if err != nil {
		slog.Error().
			Err(err).
			Msg("Failed to NewConsumer")
		return
	}
	defer cons.Close()

	low, high, err := cons.QueryWatermarkOffsets(platformTopic, 0, 5000)
	if err != nil {
		slog.Error().
			Err(err).
			Msg("Failed to QueryWatermarkOffsets")
		return
	}
	slog.Info().
		Msgf("low = %d, high = %d", low, high)

	err = cons.Assign([]kafka.TopicPartition{
		{
			Topic:     &platformTopic,
			Partition: 0,
			Offset:    kafka.Offset(high - 1),
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
			log.Info().
				Msg("ConsumePlatformConfig exiting, ctx.Done()")
			return
		default:
			msg, err := cons.ReadMessage(time.Second * 5)
			timedOut := err != nil && err.(kafka.Error).Code() == kafka.ErrTimedOut
			if err != nil && !timedOut {
				slog.Error().
					Msg("Error during ReadMessage")
			} else if !timedOut && msg != nil {
				plat := admin_pb.Platform{}
				err = proto.Unmarshal(msg.Value, &plat)
				if err != nil {
					slog.Error().
						Msg("Failed to Unmarshall Platform")
				} else {
					ch <- plat
				}
			}
		}
	}
}
