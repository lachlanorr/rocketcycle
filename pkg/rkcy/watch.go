// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/consts"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
	"github.com/lachlanorr/rocketcycle/version"
)

type watchTopic struct {
	clusterName      string
	bootstrapServers string
	topicName        string
	logLevel         zerolog.Level
}

func (wt *watchTopic) String() string {
	return fmt.Sprintf("%s__%s", wt.clusterName, wt.topicName)
}

func (wt *watchTopic) consume(ctx context.Context) {
	groupName := fmt.Sprintf("rkcy_%s__%s_watcher", platformName, wt)
	log.Info().Msgf("watching: %s", wt.topicName)

	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        wt.bootstrapServers,
		"group.id":                 groupName,
		"enable.auto.commit":       true, // librdkafka will commit to brokers for us on an interval and when we close consumer
		"enable.auto.offset.store": true, // librdkafka will commit to local store to get "at most once" behavior
	})
	if err != nil {
		log.Error().
			Err(err).
			Str("BoostrapServers", wt.bootstrapServers).
			Str("GroupId", groupName).
			Msg("Unable to kafka.NewConsumer")
		return
	}
	defer cons.Close()

	cons.Subscribe(wt.topicName, nil)

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("watchTopic.consume exiting, ctx.Done()")
			return
		default:
			msg, err := cons.ReadMessage(time.Second * 5)
			timedOut := err != nil && err.(kafka.Error).Code() == kafka.ErrTimedOut
			if err != nil && !timedOut {
				log.Error().
					Err(err).
					Msg("Error during ReadMessage")
			} else if !timedOut && msg != nil {
				log.WithLevel(wt.logLevel).
					Str("Directive", fmt.Sprintf("0x%08X", int(GetDirective(msg)))).
					Str("ReqId", GetReqId(msg)).
					Msg(wt.topicName)
				txn := pb.ApecsTxn{}
				err := proto.Unmarshal(msg.Value, &txn)
				if err == nil {
					txnJson := protojson.Format(proto.Message(&txn))
					log.WithLevel(wt.logLevel).
						Msg(txnJson)
				}
			}
		}
	}
}

func getAllWatchTopics(rtPlat *rtPlatform) []*watchTopic {
	var wts []*watchTopic
	for _, concern := range rtPlat.Concerns {
		for _, topic := range concern.Topics {
			tp, err := ParseFullTopicName(topic.CurrentTopic)
			if err == nil {
				if tp.TopicName == consts.Error || tp.TopicName == consts.Complete {
					wt := watchTopic{
						clusterName:      topic.CurrentCluster.Name,
						bootstrapServers: topic.CurrentCluster.BootstrapServers,
						topicName:        topic.CurrentTopic,
					}
					if tp.TopicName == consts.Error {
						wt.logLevel = zerolog.ErrorLevel
					} else {
						wt.logLevel = zerolog.DebugLevel
					}
					wts = append(wts, &wt)
				}
			}
		}
	}
	return wts
}

func watchResultTopics(ctx context.Context) {
	wtMap := make(map[string]bool)

	platCh := make(chan *pb.Platform)
	go consumePlatformConfig(ctx, platCh, settings.BootstrapServers, platformName)

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("watchResultTopics exiting, ctx.Done()")
			return
		case plat := <-platCh:
			rtPlat, err := newRtPlatform(plat)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to newRtPlatform")
				continue
			}

			wts := getAllWatchTopics(rtPlat)

			for _, wt := range wts {
				_, ok := wtMap[wt.String()]
				if !ok {
					wtMap[wt.String()] = true
					go wt.consume(ctx)
				}
			}
		}
	}
}

func cobraWatch(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("APECS WATCH started")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go watchResultTopics(ctx)

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	select {
	case <-interruptCh:
		log.Info().
			Msg("APECS WATCH stopped")
		cancel()
		return
	}
}
