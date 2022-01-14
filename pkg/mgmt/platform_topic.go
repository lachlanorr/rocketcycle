// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package mgmt

import (
	"context"
	"errors"
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

func PlatformReplaceFromFile(
	ctx context.Context,
	strmprov rkcy.StreamProvider,
	platform string,
	environment string,
	adminBrokers string,
	platformFilePath string,
) {
	// read platform conf file and deserialize
	platformDefJson, err := ioutil.ReadFile(platformFilePath)
	if err != nil {
		log.Fatal().
			Str("PlatformFilePath", platformFilePath).
			Err(err).
			Msg("Failed to ReadFile")
	}

	err = PlatformReplaceFromJson(
		ctx,
		strmprov,
		platform,
		environment,
		adminBrokers,
		platformDefJson,
	)

	if err != nil {
		log.Fatal().
			Str("Brokers", adminBrokers).
			Str("PlatformFilePath", platformFilePath).
			Err(err).
			Msg("Failed to PlatformReplaceFromJson")
	}
}

func PlatformReplaceFromJson(
	ctx context.Context,
	strmprov rkcy.StreamProvider,
	platform string,
	environment string,
	adminBrokers string,
	platformDefJson []byte,
) error {
	platDef := rkcypb.PlatformDef{}
	err := protojson.Unmarshal(platformDefJson, proto.Message(&platDef))
	if err != nil {
		return err
	}

	platDef.UpdateTime = timestamppb.Now()

	// create an RtPlatformDef so we run the validations that involves
	rtPlatDef, err := rkcy.NewRtPlatformDef(&platDef, platform, environment)
	if err != nil {
		return err
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
		return err
	}

	platformTopic := rkcy.PlatformTopic(platDef.Name, platDef.Environment)

	// At this point we are guaranteed to have a platform admin topic
	kafkaLogCh := make(chan kafka.LogEvent)
	go rkcy.PrintKafkaLogs(ctx, kafkaLogCh)

	prod, err := strmprov.NewProducer(adminBrokers, kafkaLogCh)
	if err != nil {
		return err
	}
	defer func() {
		prod.Close()
	}()

	msg, err := rkcy.NewKafkaMessage(&platformTopic, 0, &platDef, rkcypb.Directive_PLATFORM, telem.ExtractTraceParent(ctx))
	if err != nil {
		return err
	}

	err = prod.Produce(msg, nil)
	if err != nil {
		return err
	}

	// check channel for delivery event
	timer := time.NewTimer(10 * time.Second)
Loop:
	for {
		select {
		case <-timer.C:
			return errors.New("Timeout producing platform message")
		case ev := <-prod.Events():
			msgEv, ok := ev.(*kafka.Message)
			if ok {
				if msgEv.TopicPartition.Error != nil {
					return msgEv.TopicPartition.Error
				} else {
					break Loop
				}
			}
		}
	}
	return nil
}
