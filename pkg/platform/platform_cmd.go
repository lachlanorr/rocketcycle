// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/version"
)

func PlatformReplace(plat rkcy.Platform, platformFilePath string) {
	slog := log.With().
		Str("Brokers", plat.AdminBrokers()).
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
	rtPlatDef, err := rkcy.NewRtPlatformDef(&platDef, plat.Name(), plat.Environment())
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
	err = rkcy.CreatePlatformTopics(context.Background(), plat.AdminBrokers(), platDef.Name, platDef.Environment)
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kafkaLogCh := make(chan kafka.LogEvent)
	go rkcy.PrintKafkaLogs(ctx, kafkaLogCh)

	prod, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":  plat.AdminBrokers(),
		"acks":               -1,     // acks required from all in-sync replicas
		"message.timeout.ms": 600000, // 10 minutes

		"go.logs.channel.enable": true,
		"go.logs.channel":        kafkaLogCh,
	})
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

	msg, err := rkcy.NewKafkaMessage(&platformTopic, 0, &platDef, rkcypb.Directive_PLATFORM, rkcy.ExtractTraceParent(ctx))
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

func DecodeInstance(plat rkcy.Platform, concern string, payload64 string) {
	ctx := context.Background()

	jsonBytes, err := plat.ConcernHandlers().DecodeInstance64Json(ctx, concern, payload64)
	if err != nil {
		fmt.Printf("Failed to decode: %s\n", err.Error())
		os.Stderr.WriteString(fmt.Sprintf("Failed to decode: %s\n", err.Error()))
		os.Exit(1)
	}

	var indentJson bytes.Buffer
	err = json.Indent(&indentJson, jsonBytes, "", "  ")
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("Failed to json.Indent: %s\n", err.Error()))
		os.Exit(1)
	}

	fmt.Println(string(indentJson.Bytes()))
}

func StartApecsConsumer(
	plat rkcy.Platform,
	consumerBrokers string,
	system rkcypb.System,
	topic string,
	partition int32,
) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Str("Topic", topic).
		Int32("Partition", partition).
		Msg("APECS Consumer Started")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	defer func() {
		signal.Stop(interruptCh)
		cancel()
	}()

	bailoutCh := make(chan bool)

	var wg sync.WaitGroup

	plat.SetSystem(system)
	go startApecsRunner(
		ctx,
		plat,
		plat.AdminBrokers(),
		consumerBrokers,
		topic,
		partition,
		bailoutCh,
		&wg,
	)

	select {
	case <-interruptCh:
		log.Warn().
			Str("System", plat.System().String()).
			Str("Topic", topic).
			Int32("Partition", partition).
			Msg("APECS Consumer Stopped, SIGINT")
		cancel()
		wg.Wait()
		return
	case <-bailoutCh:
		log.Warn().
			Str("System", plat.System().String()).
			Str("Topic", topic).
			Int32("Partition", partition).
			Msg("APECS Consumer Stopped, BAILOUT")
		cancel()
		wg.Wait()
		return
	}
}
