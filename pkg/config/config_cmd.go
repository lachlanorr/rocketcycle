// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"context"
	"io/ioutil"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

func ConfigReplace(plat rkcy.Platform, configFilePath string) {
	slog := log.With().
		Str("Brokers", plat.AdminBrokers()).
		Str("ConfigFilePath", configFilePath).
		Logger()

	// read config file and deserialize
	data, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to ReadFile")
	}

	conf, err := Json2config(data)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to parseConfigFile")
	}

	// connect to kafka and make sure we have our platform topics
	err = rkcy.CreatePlatformTopics(context.Background(), plat)
	if err != nil {
		slog.Fatal().
			Err(err).
			Str("Platform", plat.Name()).
			Msg("Failed to create platform topics")
	}

	configTopic := rkcy.ConfigTopic(plat.Name(), plat.Environment())
	slog = slog.With().
		Str("Topic", configTopic).
		Logger()

	// At this point we are guaranteed to have a platform admin topic
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kafkaLogCh := make(chan kafka.LogEvent)
	go rkcy.PrintKafkaLogs(ctx, kafkaLogCh)

	prod, err := plat.NewProducer(plat.AdminBrokers(), kafkaLogCh)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to NewProducer")
	}
	defer func() {
		log.Warn().
			Str("Brokers", plat.AdminBrokers()).
			Msg("Closing kafka producer")
		prod.Close()
		log.Warn().
			Str("Brokers", plat.AdminBrokers()).
			Msg("Closed kafka producer")
	}()

	msg, err := rkcy.NewKafkaMessage(&configTopic, 0, conf, rkcypb.Directive_CONFIG_PUBLISH, rkcy.ExtractTraceParent(ctx))
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to kafkaMessage")
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
				Msg("Timeout producing config message")
		case ev := <-prod.Events():
			msgEv, ok := ev.(*kafka.Message)
			if !ok {
				slog.Warn().
					Msg("Non *kafka.Message event received from producer")
			} else {
				if msgEv.TopicPartition.Error != nil {
					slog.Warn().
						Err(msgEv.TopicPartition.Error).
						Msg("Error reported while producing config message, trying again after a delay")
					time.Sleep(1 * time.Second)
					produce()
				} else {
					slog.Info().
						Msg("Config successfully produced")
					break Loop
				}
			}
		}
	}
}
