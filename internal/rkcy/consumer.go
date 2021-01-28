// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	admin_pb "github.com/lachlanorr/rocketcycle/build/proto/admin"
)

func ConsumePlatformConfig(ctx context.Context, ch chan<- admin_pb.Platform, bootstrapServers string, platformName string) {
	platformTopic := AdminTopic(platformName)
	groupName := "__" + platformTopic + "__non_committed_group"

	slog := log.With().
		Str("BootstrapServers", bootstrapServers).
		Str("Topic", platformTopic).
		Logger()

	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  bootstrapServers,
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

	var high int64
	gotOffsets := false
	for !gotOffsets {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("ConsumePlatformConfig exiting, ctx.Done()")
			return
		default:
			_, high, err = cons.QueryWatermarkOffsets(platformTopic, 0, 5000)
			if err != nil {
				slog.Error().
					Err(err).
					Msg("Failed to QueryWatermarkOffsets, platform topic may not yet exist")
			} else {
				gotOffsets = true
			}
		}
	}

	err = cons.Assign([]kafka.TopicPartition{
		{
			Topic:     &platformTopic,
			Partition: 0,
			Offset:    kafka.Offset(Maxi(0, high-1)),
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
					Err(err).
					Msg("Error during ReadMessage")
			} else if !timedOut && msg != nil {
				plat := admin_pb.Platform{}
				if val := FindHeader(msg, "type"); val != nil {
					if string(val) == MsgTypeName(proto.Message(&plat)) {
						err = proto.Unmarshal(msg.Value, &plat)
						if err != nil {
							slog.Error().
								Err(err).
								Msg("Failed to Unmarshall Platform")
						} else {
							ch <- plat
						}
						break
					}
				}
				slog.Error().
					Err(err).
					Msg("admin topic message missing type header or header value unexpected")
			}
		}
	}
}
