// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package mgmt

import (
	"context"
	"io/ioutil"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/pkg/telem"
)

type ConfigPublishMessage struct {
	Directive rkcypb.Directive
	Timestamp time.Time
	Offset    int64
	Config    *rkcypb.Config
}

func ConsumeConfigTopic(
	ctx context.Context,
	wg *sync.WaitGroup,
	strmprov rkcy.StreamProvider,
	platform string,
	environment string,
	adminBrokers string,
	chPublish chan<- *ConfigPublishMessage,
	readyCh chan<- bool,
) {
	wg.Add(1)
	go ConsumeMgmtTopic(
		ctx,
		wg,
		strmprov,
		adminBrokers,
		rkcy.ConfigTopic(platform, environment),
		rkcypb.Directive_CONFIG,
		AT_LAST_MATCH,
		func(rawMsg *RawMessage) {
			if chPublish != nil && (rawMsg.Directive&rkcypb.Directive_CONFIG_PUBLISH) == rkcypb.Directive_CONFIG_PUBLISH {
				conf := &rkcypb.Config{}
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
	)
}

func ConfigReplace(
	ctx context.Context,
	strmprov rkcy.StreamProvider,
	platform string,
	environment string,
	adminBrokers string,
	configFilePath string,
) {
	slog := log.With().
		Str("Platform", platform).
		Str("Environment", environment).
		Str("Brokers", adminBrokers).
		Str("ConfigFilePath", configFilePath).
		Logger()

	// read config file and deserialize
	data, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to ReadFile")
	}

	conf, err := rkcy.JsonToConfig(data)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to parseConfigFile")
	}

	// connect to kafka and make sure we have our platform topics
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	err = rkcy.CreatePlatformTopics(
		ctx,
		strmprov,
		platform,
		environment,
		adminBrokers,
	)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to create platform topics")
	}

	configTopic := rkcy.ConfigTopic(platform, environment)
	slog = slog.With().
		Str("Topic", configTopic).
		Logger()

	// At this point we are guaranteed to have our platform topics
	kafkaLogCh := make(chan kafka.LogEvent)
	go rkcy.PrintKafkaLogs(ctx, kafkaLogCh)

	prod, err := strmprov.NewProducer(adminBrokers, kafkaLogCh)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to NewProducer")
	}
	defer func() {
		slog.Trace().
			Msg("Closing kafka producer")
		prod.Close()
		slog.Trace().
			Msg("Closed kafka producer")
	}()

	msg, err := rkcy.NewKafkaMessage(&configTopic, 0, conf, rkcypb.Directive_CONFIG_PUBLISH, telem.ExtractTraceParent(ctx))
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
