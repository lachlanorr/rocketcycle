// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rkcy/pkg/rkcy/pb"
)

const undefinedPlatformName string = "__UNDEFINED__"

var platformName string = undefinedPlatformName

func initPlatformName(name string) {
	if platformName != undefinedPlatformName {
		panic("Platform can be initialized only once, current name: " + platformName)
	}
	platformName = name
}

func PlatformName() string {
	return platformName
}

// Platform pb, with some convenience lookup maps
type rtPlatform struct {
	Platform *pb.Platform
	Hash     string
	Concerns map[string]*rtConcern
	Clusters map[string]*pb.Platform_Cluster
}

type rtConcern struct {
	Concern *pb.Platform_Concern
	Topics  map[string]*rtTopics
}

type rtTopics struct {
	Topics         *pb.Platform_Concern_Topics
	CurrentTopic   string
	CurrentCluster *pb.Platform_Cluster
	FutureTopic    string
	FutureCluster  *pb.Platform_Cluster
}

func newRtConcern(rtPlat *rtPlatform, concern *pb.Platform_Concern) (*rtConcern, error) {
	rtConc := rtConcern{
		Concern: concern,
		Topics:  make(map[string]*rtTopics),
	}
	for _, topics := range concern.Topics {
		// verify topics only appear once
		if _, ok := rtConc.Topics[topics.Name]; ok {
			return nil, fmt.Errorf("Topic '%s' appears more than once in Concern '%s' definition", topics.Name, rtConc.Concern.Name)
		}
		rtTops, err := newRtTopics(rtPlat, &rtConc, topics)
		if err != nil {
			return nil, err
		}
		rtConc.Topics[topics.Name] = rtTops
	}
	return &rtConc, nil
}

func newRtTopics(rtPlat *rtPlatform, rtConc *rtConcern, topics *pb.Platform_Concern_Topics) (*rtTopics, error) {
	rtTops := rtTopics{
		Topics: topics,
	}

	pref := BuildTopicNamePrefix(rtPlat.Platform.Name, rtConc.Concern.Name, rtConc.Concern.Type)
	var ok bool

	rtTops.CurrentTopic = BuildTopicName(pref, topics.Name, topics.Current.Generation)
	rtTops.CurrentCluster, ok = rtPlat.Clusters[topics.Current.ClusterName]
	if !ok {
		return nil, fmt.Errorf("Topic '%s' has invalid Current ClusterName '%s'", topics.Name, topics.Current.ClusterName)
	}

	if topics.Future != nil {
		rtTops.FutureTopic = BuildTopicName(pref, topics.Name, topics.Future.Generation)
		rtTops.FutureCluster, ok = rtPlat.Clusters[topics.Future.ClusterName]
		if !ok {
			return nil, fmt.Errorf("Topic '%s' has invalid Future ClusterName '%s'", topics.Name, topics.Future.ClusterName)
		}
	}
	return &rtTops, nil
}

func initTopic(topic *pb.Platform_Concern_Topic, defaultCluster string) *pb.Platform_Concern_Topic {
	if topic == nil {
		topic = &pb.Platform_Concern_Topic{}
	}

	if topic.Generation <= 0 {
		topic.Generation = 1
	}
	if topic.ClusterName == "" {
		topic.ClusterName = defaultCluster
	}
	if topic.PartitionCount <= 0 {
		topic.PartitionCount = 1
	} else if topic.PartitionCount > 1024 {
		topic.PartitionCount = 1024
	}

	return topic
}

func initTopics(topics *pb.Platform_Concern_Topics, defaultCluster string, concernType pb.Platform_Concern_Type) *pb.Platform_Concern_Topics {
	if topics == nil {
		topics = &pb.Platform_Concern_Topics{}
	}

	topics.Current = initTopic(topics.Current, defaultCluster)
	if topics.Future != nil {
		topics.Future = initTopic(topics.Future, defaultCluster)
	}

	if concernType == pb.Platform_Concern_APECS {
		if topics.ConsumerProgram == nil {
			switch topics.Name {
			case "process":
				topics.ConsumerProgram = &pb.Platform_Concern_Program{
					Name: "@platform",
					Args: []string{"process", "-t", "@topic", "-p", "@partition"},
				}
			case "storage":
				topics.ConsumerProgram = &pb.Platform_Concern_Program{
					Name: "@platform",
					Args: []string{"storage", "-t", "@topic", "-p", "@partition"},
				}
			}
		}
	}

	return topics
}

func newRtPlatform(platform *pb.Platform) (*rtPlatform, error) {
	if platform.Name != PlatformName() {
		return nil, fmt.Errorf("Platform Name mismatch, '%s' != '%s'", platform.Name, PlatformName)
	}

	rtPlat := rtPlatform{
		Platform: platform,
		Concerns: make(map[string]*rtConcern),
		Clusters: make(map[string]*pb.Platform_Cluster),
	}

	platJson := protojson.Format(proto.Message(rtPlat.Platform))
	sha256Bytes := sha256.Sum256([]byte(platJson))
	rtPlat.Hash = hex.EncodeToString(sha256Bytes[:])

	if len(rtPlat.Platform.Clusters) <= 0 {
		return nil, fmt.Errorf("No clusters defined")
	}
	for idx, cluster := range rtPlat.Platform.Clusters {
		if cluster.Name == "" {
			return nil, fmt.Errorf("Cluster %d missing name field", idx)
		}
		if cluster.BootstrapServers == "" {
			return nil, fmt.Errorf("Cluster '%s' missing bootstrap_servers field", cluster.Name)
		}
		// verify clusters only appear once
		if _, ok := rtPlat.Clusters[cluster.Name]; ok {
			return nil, fmt.Errorf("Cluster '%s' appears more than once in Platform '%s' definition", cluster.Name, rtPlat.Platform.Name)
		}
		rtPlat.Clusters[cluster.Name] = cluster
	}

	requiredTopics := map[pb.Platform_Concern_Type][]string{
		pb.Platform_Concern_GENERAL: {"admin", "error"},
		pb.Platform_Concern_BATCH:   {"admin", "error"},
		pb.Platform_Concern_APECS:   {"admin", "process", "error", "complete", "storage"},
	}

	for idx, concern := range rtPlat.Platform.Concerns {
		if concern.Name == "" {
			return nil, fmt.Errorf("Concern %d missing name field", idx)
		}

		defaultCluster := ""
		var topicNames []string
		// build list of topicNames for validation steps below
		for _, topics := range concern.Topics {
			topicNames = append(topicNames, topics.Name)
			if defaultCluster == "" && topics.Current != nil {
				defaultCluster = topics.Current.ClusterName
			}
		}

		// if we still don't have a defaultCluster, choose the first one
		if defaultCluster == "" {
			defaultCluster = rtPlat.Platform.Clusters[0].Name
		}

		// validate our expected required topics are there, add any with defaults if not present
		for _, req := range requiredTopics[concern.Type] {
			if !contains(topicNames, req) {
				// conern.Topics will get initialized with reasonable defaults during topic validation below
				concern.Topics = append(concern.Topics, &pb.Platform_Concern_Topics{Name: req})
			}
		}

		// validate all topics definitions
		for idx, _ := range concern.Topics {
			concern.Topics[idx] = initTopics(concern.Topics[idx], defaultCluster, concern.Type)
			if err := validateTopics(concern.Topics[idx], rtPlat.Clusters); err != nil {
				return nil, fmt.Errorf("Concern '%s' has invalid '%s' Topics: %s", concern.Name, concern.Topics[idx].Name, err.Error())
			}
		}

		// verify concerns only appear once
		if _, ok := rtPlat.Concerns[concern.Name]; ok {
			return nil, fmt.Errorf("Concern '%s' appears more than once in Platform '%s' definition", concern.Name, rtPlat.Platform.Name)
		}
		rtConc, err := newRtConcern(&rtPlat, concern)
		if err != nil {
			return nil, err
		}
		rtPlat.Concerns[concern.Name] = rtConc
	}

	return &rtPlat, nil
}

func validateTopics(topics *pb.Platform_Concern_Topics, clusters map[string]*pb.Platform_Cluster) error {
	if topics.Name == "" {
		return errors.New("Topics missing Name field")
	}
	// admin topics are special and have stricter rules
	if topics.Name == "admin" {
		if topics.Current == nil || topics.Future != nil {
			return fmt.Errorf("'admin' Topics only exist as current and not future")
		}
		if topics.Current.PartitionCount != 1 {
			return fmt.Errorf("'admin' Topics must have exactly 1 current partition")
		}
	} else {
		if topics.Current == nil {
			return errors.New("Topics missing Current Topic")
		} else {
			if err := validateTopic(topics.Current, clusters); err != nil {
				return err
			}
		}
		if topics.Future != nil {
			if err := validateTopic(topics.Future, clusters); err != nil {
				return err
			}
			if topics.Current.Generation != topics.Future.Generation+1 {
				return errors.New("Future generation not Current + 1")
			}
		}
	}
	if topics.ConsumerProgram != nil {
		if topics.ConsumerProgram.Name == "" {
			return errors.New("Command cannot have blank Name")
		}
	}
	return nil
}

func validateTopic(topic *pb.Platform_Concern_Topic, clusters map[string]*pb.Platform_Cluster) error {
	if topic.Generation == 0 {
		return errors.New("Topic missing Generation field")
	}
	if topic.ClusterName == "" {
		return errors.New("Topic missing ClusterName field")
	}
	if _, ok := clusters[topic.ClusterName]; !ok {
		return fmt.Errorf("Topic refers to non-existent cluster: '%s'", topic.ClusterName)
	}
	if topic.PartitionCount < 1 || topic.PartitionCount > 1024 {
		return fmt.Errorf("Topic with out of bounds PartitionCount %d", topic.PartitionCount)
	}
	return nil
}

func uncommittedGroupName(topic string, partition int) string {
	return fmt.Sprintf("__%s_%d__non_comitted_group", topic, partition)
}

func consumePlatformConfig(ctx context.Context, ch chan<- *pb.Platform, bootstrapServers string, platformName string) {
	platformTopic := adminTopic(platformName)
	groupName := uncommittedGroupName(platformTopic, 0)

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
			Offset:    kafka.Offset(maxi64(0, high-1)),
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
				plat := pb.Platform{}
				if val := findHeader(msg, "type"); val != nil {
					if string(val) == msgTypeName(proto.Message(&plat)) {
						err = proto.Unmarshal(msg.Value, &plat)
						if err != nil {
							slog.Error().
								Err(err).
								Msg("Failed to Unmarshall Platform")
						} else {
							ch <- &plat
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

func cobraPlatUpdate(cmd *cobra.Command, args []string) {
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
