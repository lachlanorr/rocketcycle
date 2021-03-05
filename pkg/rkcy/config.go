// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"io/ioutil"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rkcy/pkg/rkcy/pb"
)

func confCommand(cmd *cobra.Command, args []string) {
	slog := log.With().
		Str("BootstrapServers", settings.BootstrapServers).
		Str("ConfigPath", settings.ConfigFilePath).
		Logger()

	// read platform conf file and deserialize
	conf, err := ioutil.ReadFile(settings.ConfigFilePath)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to ReadFile")
	}
	plat := pb.Platform{}
	err = protojson.Unmarshal(conf, proto.Message(&plat))
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to unmarshal platform")
	}
	platMar, err := proto.Marshal(&plat)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to Marshal platform")
	}

	// create an rtPlatform so we run the validations that involves
	rtPlat, err := newRtPlatform(&plat)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to create newRtPlatform")
	}
	jsonBytes, _ := protojson.Marshal(proto.Message(rtPlat.Platform))
	log.Info().
		Str("PlatformJson", string(jsonBytes)).
		Msg("Platform parsed")

	// connect to kafka and make sure we have our platform topic
	adminTopic, err := createAdminTopic(context.Background(), settings.BootstrapServers, plat.Name)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msgf("Failed to createAdminTopic for platform %s", plat.Name)
	}
	slog = slog.With().
		Str("Topic", adminTopic).
		Logger()
	slog.Info().
		Msgf("Created platform admin topic: %s", adminTopic)

	// At this point we are guaranteed to have a platform admin topic
	prod, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": settings.BootstrapServers})
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to NewProducer")
	}
	defer prod.Close()

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &adminTopic, Partition: 0},
		Value:          platMar,
		Headers:        msgTypeHeaders(proto.Message(&plat)),
	}

	produce := func() {
		err := prod.Produce(msg, nil)
		if err != nil {
			slog.Fatal().
				Err(err).
				Msg("Failed to Produce")
		}
	}

	produce()

	// check channel for delivery event
	timer := time.NewTimer(10 * time.Second)
Loop:
	for {
		select {
		case <-timer.C:
			slog.Fatal().
				Msg("Timeout producing platform message")
			break Loop
		case ev := <-prod.Events():
			msgEv, ok := ev.(*kafka.Message)
			if !ok {
				slog.Warn().
					Msg("Non *kafka.Message event received from producer")
			} else {
				if msgEv.TopicPartition.Error != nil {
					slog.Warn().
						Err(msgEv.TopicPartition.Error).
						Msg("Error reported while producing platform message, trying again after a delay")
					time.Sleep(1 * time.Second)
					produce()
				} else {
					slog.Info().
						Msg("Platform config successfully produced")
					break Loop
				}
			}
		}
	}
}
