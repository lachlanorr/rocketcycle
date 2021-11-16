// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	otel_codes "go.opentelemetry.io/otel/codes"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

const kUndefinedPlatformName string = "__UNDEFINED__"

var gPlatformName string = kUndefinedPlatformName
var gEnvironment string

var nameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z\-]{1,15}$`)

func IsValidName(name string) bool {
	return nameRe.MatchString(name)
}

func initPlatformName(name string) {
	if gPlatformName != kUndefinedPlatformName {
		panic("Platform can be initialized only once, current name: " + gPlatformName)
	}
	gPlatformName = name
	if !IsValidName(gPlatformName) {
		log.Fatal().Msgf("Invalid platform name: %s", gPlatformName)
	}

	gEnvironment = os.Getenv("RKCY_ENVIRONMENT")
	if !IsValidName(gEnvironment) {
		log.Fatal().Msgf("Invalid RKCY_ENVIRONMENT: %s", gEnvironment)
	}
}

func PlatformName() string {
	return gPlatformName
}

func Environment() string {
	return gEnvironment
}

// Platform pb, with some convenience lookup maps

type rtPlatform struct {
	Platform             *Platform
	Hash                 string
	Concerns             map[string]*rtConcern
	Clusters             map[string]*Platform_Cluster
	AdminCluster         string
	StorageTargets       map[string]*Platform_StorageTarget
	PrimaryStorageTarget string
}

type rtConcern struct {
	Concern *Platform_Concern
	Topics  map[string]*rtTopics
}

type rtTopics struct {
	Topics                     *Platform_Concern_Topics
	CurrentTopic               string
	CurrentTopicPartitionCount int32
	CurrentCluster             *Platform_Cluster
	FutureTopic                string
	FutureTopicPartitionCount  int32
	FutureCluster              *Platform_Cluster
}

func newRtConcern(rtPlat *rtPlatform, concern *Platform_Concern) (*rtConcern, error) {
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

func isACETopic(topic string) bool {
	return topic == string(ADMIN) || topic == string(ERROR) || topic == string(COMPLETE)
}

func newRtTopics(rtPlat *rtPlatform, rtConc *rtConcern, topics *Platform_Concern_Topics) (*rtTopics, error) {
	rtTops := rtTopics{
		Topics: topics,
	}

	pref := BuildTopicNamePrefix(rtPlat.Platform.Name, rtPlat.Platform.Environment, rtConc.Concern.Name, rtConc.Concern.Type)
	var ok bool

	rtTops.CurrentTopic = BuildTopicName(pref, topics.Name, topics.Current.Generation)
	rtTops.CurrentTopicPartitionCount = maxi32(1, topics.Current.PartitionCount)
	rtTops.CurrentCluster, ok = rtPlat.Clusters[topics.Current.Cluster]
	if !ok {
		return nil, fmt.Errorf("Topic '%s.%s' has invalid Current Cluster '%s'", rtConc.Concern.Name, topics.Name, topics.Current.Cluster)
	}

	if topics.Future != nil {
		rtTops.FutureTopic = BuildTopicName(pref, topics.Name, topics.Future.Generation)
		rtTops.FutureTopicPartitionCount = maxi32(1, topics.Future.PartitionCount)
		rtTops.FutureCluster, ok = rtPlat.Clusters[topics.Future.Cluster]
		if !ok {
			return nil, fmt.Errorf("Topic '%s.%s' has invalid Future Cluster '%s'", rtConc.Concern.Name, topics.Name, topics.Future.Cluster)
		}
	}

	if isACETopic(rtTops.Topics.Name) {
		if rtTops.CurrentTopicPartitionCount != 1 {
			return nil, fmt.Errorf("Topic '%s.%s' has invalid partition count current=%d", rtConc.Concern.Name, topics.Name, rtTops.CurrentTopicPartitionCount)
		}
	}

	return &rtTops, nil
}

func initTopic(topic *Platform_Concern_Topic, adminCluster string) *Platform_Concern_Topic {
	if topic == nil {
		topic = &Platform_Concern_Topic{}
	}

	if topic.Generation <= 0 {
		topic.Generation = 1
	}
	if topic.Cluster == "" {
		topic.Cluster = adminCluster
	}
	if topic.PartitionCount <= 0 {
		topic.PartitionCount = 1
	} else if topic.PartitionCount > MAX_PARTITION {
		topic.PartitionCount = MAX_PARTITION
	}

	return topic
}

func initTopics(
	topics *Platform_Concern_Topics,
	adminCluster string,
	concernType Platform_Concern_Type,
	storageTargets []*Platform_StorageTarget,
) *Platform_Concern_Topics {
	if topics == nil {
		topics = &Platform_Concern_Topics{}
	}

	topics.Current = initTopic(topics.Current, adminCluster)
	if topics.Future != nil {
		topics.Future = initTopic(topics.Future, adminCluster)
	}

	if concernType == Platform_Concern_APECS {
		topics.ConsumerPrograms = nil
		switch topics.Name {
		case "process":
			prog := &Program{
				Name:   "./@platform",
				Args:   []string{"process", "--otelcol_endpoint", "@otelcol_endpoint", "--admin_brokers", "@admin_brokers", "--consumer_brokers", "@consumer_brokers", "-t", "@topic", "-p", "@partition"},
				Abbrev: "p/@concern/@partition",
			}
			topics.ConsumerPrograms = append(topics.ConsumerPrograms, prog)
		case "storage":
			for _, stgTgt := range storageTargets {
				if stgTgt.IsPrimary {
					prog := &Program{
						Name: "./@platform",
						Args: []string{"storage", "--otelcol_endpoint", "@otelcol_endpoint", "--admin_brokers", "@admin_brokers", "--consumer_brokers", "@consumer_brokers", "-t", "@topic", "-p", "@partition", "--storage_target", stgTgt.Name},
					}
					prog.Abbrev = fmt.Sprintf("s/*%s/@concern/@partition", stgTgt.Name)
					topics.ConsumerPrograms = append(topics.ConsumerPrograms, prog)
				}
			}
		case "storage-scnd":
			for _, stgTgt := range storageTargets {
				if !stgTgt.IsPrimary {
					prog := &Program{
						Name: "./@platform",
						Args: []string{"storage-scnd", "--otelcol_endpoint", "@otelcol_endpoint", "--admin_brokers", "@admin_brokers", "--consumer_brokers", "@consumer_brokers", "-t", "@topic", "-p", "@partition", "--storage_target", stgTgt.Name},
					}
					prog.Abbrev = fmt.Sprintf("s/%s/@concern/@partition", stgTgt.Name)
					topics.ConsumerPrograms = append(topics.ConsumerPrograms, prog)
				}
			}
		}
	}

	return topics
}

func newRtPlatform(platform *Platform) (*rtPlatform, error) {
	if platform.Name != PlatformName() {
		return nil, fmt.Errorf("Platform Name mismatch, '%s' != '%s'", platform.Name, PlatformName())
	}
	if platform.Environment != Environment() {
		return nil, fmt.Errorf("Environment mismatch, '%s' != '%s'", platform.Environment, Environment())
	}

	rtPlat := rtPlatform{
		Platform:       platform,
		Concerns:       make(map[string]*rtConcern),
		Clusters:       make(map[string]*Platform_Cluster),
		StorageTargets: make(map[string]*Platform_StorageTarget),
	}

	platJson := protojson.Format(proto.Message(rtPlat.Platform))
	sha256Bytes := sha256.Sum256([]byte(platJson))
	rtPlat.Hash = hex.EncodeToString(sha256Bytes[:])

	if !rtPlat.Platform.UpdateTime.IsValid() {
		return nil, fmt.Errorf("Invalid UpdateTime: %s", rtPlat.Platform.UpdateTime.AsTime())
	}

	if len(rtPlat.Platform.Clusters) <= 0 {
		return nil, fmt.Errorf("No clusters defined")
	}
	for idx, cluster := range rtPlat.Platform.Clusters {
		if cluster.Name == "" {
			return nil, fmt.Errorf("Cluster %d missing name field", idx)
		}
		if cluster.Brokers == "" {
			return nil, fmt.Errorf("Cluster '%s' missing brokers field", cluster.Name)
		}
		if cluster.IsAdmin {
			if rtPlat.AdminCluster != "" {
				return nil, fmt.Errorf("More than one admin cluster")
			}
			rtPlat.AdminCluster = cluster.Name
		}
		// verify clusters only appear once
		if _, ok := rtPlat.Clusters[cluster.Name]; ok {
			return nil, fmt.Errorf("Cluster '%s' appears more than once in Platform '%s' definition", cluster.Name, rtPlat.Platform.Name)
		}
		rtPlat.Clusters[cluster.Name] = cluster
	}
	if rtPlat.AdminCluster == "" {
		return nil, fmt.Errorf("No admin cluster defined")
	}

	if len(rtPlat.Platform.StorageTargets) <= 0 {
		return nil, fmt.Errorf("No storage targets defined")
	}
	for idx, sttg := range rtPlat.Platform.StorageTargets {
		if sttg.Name == "" {
			return nil, fmt.Errorf("Storage target %d missing name field", idx)
		}
		if sttg.Type == "" {
			return nil, fmt.Errorf("Storage target '%s' missing type field", sttg.Name)
		}
		if sttg.IsPrimary {
			if rtPlat.PrimaryStorageTarget != "" {
				return nil, fmt.Errorf("More than one primary storage target")
			}
			rtPlat.PrimaryStorageTarget = sttg.Name
		}
		// verify clusters only appear once
		if _, ok := rtPlat.StorageTargets[sttg.Name]; ok {
			return nil, fmt.Errorf("Storage target '%s' appears more than once in Platform '%s' definition", sttg.Name, rtPlat.Platform.Name)
		}
		rtPlat.StorageTargets[sttg.Name] = sttg
	}
	if rtPlat.PrimaryStorageTarget == "" {
		return nil, fmt.Errorf("No primary storage target defined")
	}

	requiredTopics := map[Platform_Concern_Type][]string{
		Platform_Concern_GENERAL: {"admin", "error"},
		Platform_Concern_BATCH:   {"admin", "error"},
		Platform_Concern_APECS:   {"admin", "process", "error", "complete", "storage", "storage-scnd"},
	}

	for idx, concern := range rtPlat.Platform.Concerns {
		if concern.Name == "" {
			return nil, fmt.Errorf("Concern %d missing name field", idx)
		}

		var topicNames []string
		// build list of topicNames for validation steps below
		for _, topics := range concern.Topics {
			topicNames = append(topicNames, topics.Name)
		}

		// validate our expected required topics are there, add any with defaults if not present
		for _, req := range requiredTopics[concern.Type] {
			if !contains(topicNames, req) {
				// conern.Topics will get initialized with reasonable defaults during topic validation below
				concern.Topics = append(concern.Topics, &Platform_Concern_Topics{Name: req})
			}
		}

		// ensure APECS concern only has required topics
		if concern.Type == Platform_Concern_APECS {
			// simple len check is adequate since we added all required above
			if len(requiredTopics[concern.Type]) != len(concern.Topics) {
				return nil, fmt.Errorf("ApecsConcern %d contains invalid command %+v required vs %+v total", idx, requiredTopics, concern.Topics)
			}
		}

		// validate all topics definitions
		for idx, _ := range concern.Topics {
			concern.Topics[idx] = initTopics(concern.Topics[idx], rtPlat.AdminCluster, concern.Type, rtPlat.Platform.StorageTargets)
			if err := validateTopics(concern, concern.Topics[idx], rtPlat.Clusters); err != nil {
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

var singlePartitionTopics = map[string]bool{
	"admin":    true,
	"error":    true,
	"complete": true,
}

func validateTopics(concern *Platform_Concern, topics *Platform_Concern_Topics, clusters map[string]*Platform_Cluster) error {
	if topics.Name == "" {
		return fmt.Errorf("Topics missing Name field: %s", topics.Name)
	}

	if singlePartitionTopics[topics.Name] {
		if topics.Current != nil && topics.Current.PartitionCount != 1 {
			return fmt.Errorf("'%s' Topics must have exactly 1 partition", topics.Name)
		}
		if topics.Future != nil && topics.Future.PartitionCount != 1 {
			return fmt.Errorf("'%s' Topics must have exactly 1 partition", topics.Name)
		}
	}

	if topics.Current == nil {
		return fmt.Errorf("Topics missing Current Topic: %s", topics.Name)
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
			return fmt.Errorf("Future generation not Current + 1")
		}
	}

	for idx, consProg := range topics.ConsumerPrograms {
		if consProg != nil {
			if consProg.Name == "" {
				return fmt.Errorf("Program cannot have blank Name: %d", idx)
			}
			if consProg.Abbrev == "" {
				return fmt.Errorf("Program cannot have blank Abbrev: %s", consProg.Name)
			}
		}
	}
	return nil
}

func validateTopic(topic *Platform_Concern_Topic, clusters map[string]*Platform_Cluster) error {
	if topic.Generation == 0 {
		return fmt.Errorf("Topic missing Generation field")
	}
	if topic.Cluster == "" {
		return fmt.Errorf("Topic missing Cluster field")
	}
	if _, ok := clusters[topic.Cluster]; !ok {
		return fmt.Errorf("Topic refers to non-existent cluster: '%s'", topic.Cluster)
	}
	if topic.PartitionCount < 1 || topic.PartitionCount > MAX_PARTITION {
		return fmt.Errorf("Topic with out of bounds PartitionCount %d", topic.PartitionCount)
	}
	return nil
}

func uncommittedGroupName(topic string, partition int) string {
	return fmt.Sprintf("__%s_%d__non_comitted_group", topic, partition)
}

func uncommittedGroupNameAllPartitions(topic string) string {
	return fmt.Sprintf("__%s_ALL__non_comitted_group", topic)
}

func cobraPlatReplace(cmd *cobra.Command, args []string) {
	ctx, span := Telem().StartFunc(context.Background())
	defer span.End()

	slog := log.With().
		Str("Brokers", gSettings.AdminBrokers).
		Str("PlatformPath", gSettings.PlatformFilePath).
		Logger()

	// read platform conf file and deserialize
	conf, err := ioutil.ReadFile(gSettings.PlatformFilePath)
	if err != nil {
		span.SetStatus(otel_codes.Error, err.Error())
		slog.Fatal().
			Err(err).
			Msg("Failed to ReadFile")
	}
	plat := Platform{}
	err = protojson.Unmarshal(conf, proto.Message(&plat))
	if err != nil {
		span.SetStatus(otel_codes.Error, err.Error())
		slog.Fatal().
			Err(err).
			Msg("Failed to unmarshal platform")
	}

	plat.UpdateTime = timestamppb.Now()

	// create an rtPlatform so we run the validations that involves
	rtPlat, err := newRtPlatform(&plat)
	if err != nil {
		span.SetStatus(otel_codes.Error, err.Error())
		slog.Fatal().
			Err(err).
			Msg("Failed to newRtPlatform")
	}
	jsonBytes, _ := protojson.Marshal(proto.Message(rtPlat.Platform))
	log.Info().
		Str("PlatformJson", string(jsonBytes)).
		Msg("Platform parsed")

	// connect to kafka and make sure we have our platform topics
	err = createPlatformTopics(context.Background(), gSettings.AdminBrokers, plat.Name, plat.Environment)
	if err != nil {
		span.SetStatus(otel_codes.Error, err.Error())
		slog.Fatal().
			Err(err).
			Str("Platform", plat.Name).
			Msg("Failed to create platform topics")
	}

	platformTopic := PlatformTopic(plat.Name, plat.Environment)
	slog = slog.With().
		Str("Topic", platformTopic).
		Logger()

	// At this point we are guaranteed to have a platform admin topic
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kafkaLogCh := make(chan kafka.LogEvent)
	go printKafkaLogs(ctx, kafkaLogCh)

	prod, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":  gSettings.AdminBrokers,
		"acks":               -1,     // acks required from all in-sync replicas
		"message.timeout.ms": 600000, // 10 minutes

		"go.logs.channel.enable": true,
		"go.logs.channel":        kafkaLogCh,
	})
	if err != nil {
		span.SetStatus(otel_codes.Error, err.Error())
		slog.Fatal().
			Err(err).
			Msg("Failed to NewProducer")
	}
	defer func() {
		log.Warn().
			Str("Brokers", gSettings.AdminBrokers).
			Msg("Closing kafka producer")
		prod.Close()
		log.Warn().
			Str("Brokers", gSettings.AdminBrokers).
			Msg("Closed kafka producer")
	}()

	msg, err := newKafkaMessage(&platformTopic, 0, &plat, Directive_PLATFORM, ExtractTraceParent(ctx))
	if err != nil {
		span.SetStatus(otel_codes.Error, err.Error())
		slog.Fatal().
			Err(err).
			Msg("Failed to kafkaMessage")
	}

	produce := func() {
		err := prod.Produce(msg, nil)
		if err != nil {
			span.SetStatus(otel_codes.Error, err.Error())
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
