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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/pkg/telem"
)

type PlatformMessage struct {
	Directive    rkcypb.Directive
	Timestamp    time.Time
	Offset       int64
	NewRtPlatDef *rkcy.RtPlatformDef
	OldRtPlatDef *rkcy.RtPlatformDef
}

func ConsumePlatformTopic(
	ctx context.Context,
	wg *sync.WaitGroup,
	strmprov rkcy.StreamProvider,
	platform string,
	environment string,
	adminBrokers string,
	ch chan<- *PlatformMessage,
	readyCh chan<- bool,
) {
	var currRtPlatDef *rkcy.RtPlatformDef

	wg.Add(1)
	go ConsumeMgmtTopic(
		ctx,
		wg,
		strmprov,
		adminBrokers,
		rkcy.PlatformTopic(platform, environment),
		rkcypb.Directive_PLATFORM,
		AT_LAST_MATCH,
		func(rawMsg *RawMessage) {
			platDef := &rkcypb.PlatformDef{}
			err := proto.Unmarshal(rawMsg.Value, platDef)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to Unmarshal Platform")
				return
			}

			rtPlatDef, err := rkcy.NewRtPlatformDef(platDef, platform, environment)
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
	)
}

func PlatformReplace(
	ctx context.Context,
	strmprov rkcy.StreamProvider,
	platform string,
	environment string,
	adminBrokers string,
	platformFilePath string,
) {
	slog := log.With().
		Str("Brokers", adminBrokers).
		Str("PlatformFilePath", platformFilePath).
		Logger()

	// read platform conf file and deserialize
	conf, err := ioutil.ReadFile(platformFilePath)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to ReadFile")
	}
	platDef := rkcypb.PlatformDef{}
	err = protojson.Unmarshal(conf, proto.Message(&platDef))
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to unmarshal platform")
	}

	platDef.UpdateTime = timestamppb.Now()

	// create an RtPlatformDef so we run the validations that involves
	rtPlatDef, err := rkcy.NewRtPlatformDef(&platDef, platform, environment)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to newRtPlatform")
	}
	jsonBytes, _ := protojson.Marshal(proto.Message(rtPlatDef.PlatformDef))
	log.Info().
		Str("PlatformJson", string(jsonBytes)).
		Msg("Platform parsed")

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
			Str("Platform", platDef.Name).
			Msg("Failed to create platform topics")
	}

	platformTopic := rkcy.PlatformTopic(platDef.Name, platDef.Environment)
	slog = slog.With().
		Str("Topic", platformTopic).
		Logger()

	// At this point we are guaranteed to have a platform admin topic
	kafkaLogCh := make(chan kafka.LogEvent)
	go rkcy.PrintKafkaLogs(ctx, kafkaLogCh)

	prod, err := strmprov.NewProducer(adminBrokers, kafkaLogCh)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to NewProducer")
	}
	defer func() {
		log.Warn().
			Str("Brokers", adminBrokers).
			Msg("Closing kafka producer")
		prod.Close()
		log.Warn().
			Str("Brokers", adminBrokers).
			Msg("Closed kafka producer")
	}()

	msg, err := rkcy.NewKafkaMessage(&platformTopic, 0, &platDef, rkcypb.Directive_PLATFORM, telem.ExtractTraceParent(ctx))
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
				Msg("Timeout producing platform message")
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
