// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	admin_pb "github.com/lachlanorr/rocketcycle/build/proto/admin"
	"github.com/lachlanorr/rocketcycle/internal/rckafka"
)

// Platform pb, with some convenience lookup maps
type runtimePlatform struct {
	platform *admin_pb.Platform
	hash     string
	apps     map[string]*admin_pb.Platform_App
	clusters map[string]*admin_pb.Platform_Cluster
}

var exists struct{}
var oldRtPlat *runtimePlatform = nil

type clusterInfo struct {
	cluster        *admin_pb.Platform_Cluster
	admin          *kafka.AdminClient
	existingTopics map[string]struct{}
	brokerCount    int
}

func (ci *clusterInfo) Close() {
	ci.admin.Close()
}

func createTopic(ci *clusterInfo, name string, numPartitions int) error {
	replicationFactor := min(3, ci.brokerCount)

	topicSpec := []kafka.TopicSpecification{
		{
			Topic:             name,
			NumPartitions:     numPartitions,
			ReplicationFactor: replicationFactor,
		},
	}

	timeout, _ := time.ParseDuration("30s")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := ci.admin.CreateTopics(ctx, topicSpec, nil)
	if err != nil {
		return fmt.Errorf("Unable to create topic: %s", name)
	}

	var errs []string
	for _, res := range result {
		if res.Error.Code() != kafka.ErrNoError {
			errs = append(errs, fmt.Sprintf("createTopic error for topic %s: %s", res.Topic, res.Error.Error()))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	log.Info().
		Str("ClusterName", ci.cluster.Name).
		Str("Topic", name).
		Int("NumPartitions", numPartitions).
		Int("ReplicationFactor", replicationFactor).
		Msg("Topic created")

	return nil
}

func NewClusterInfo(cluster *admin_pb.Platform_Cluster) (*clusterInfo, error) {
	var ci = clusterInfo{}

	config := make(kafka.ConfigMap)
	config.SetKey("bootstrap.servers", cluster.BootstrapServers)

	var err error
	ci.admin, err = kafka.NewAdminClient(&config)
	if err != nil {
		return nil, err
	}
	ci.cluster = cluster

	ci.existingTopics = make(map[string]struct{})

	md, err := ci.admin.GetMetadata(nil, true, 1000)

	if err != nil {
		defer ci.admin.Close()
		return nil, err
	}

	sortedTopics := make([]string, 0, len(md.Topics))
	ci.brokerCount = len(md.Brokers)
	for _, tp := range md.Topics {
		sortedTopics = append(sortedTopics, tp.Topic)
		ci.existingTopics[tp.Topic] = exists
	}

	sort.Strings(sortedTopics)
	for _, topicName := range sortedTopics {
		log.Info().
			Str("ClusterName", cluster.Name).
			Str("Topic", topicName).
			Msg("Topic found")
	}

	return &ci, nil
}

func buildTopicNamePrefix(platformName string, appName string, appType admin_pb.Platform_App_Type) string {
	return fmt.Sprintf("rc.%s.%s.%s", platformName, appName, admin_pb.Platform_App_Type_name[int32(appType)])
}

func buildTopicName(topicNamePrefix string, name string, createdAt int64) string {
	return fmt.Sprintf("%s.%s.%010d", topicNamePrefix, name, createdAt)
}

func FindApp(platform *admin_pb.Platform, appName string) *admin_pb.Platform_App {
	for _, app := range platform.Apps {
		if app.Name == appName {
			return app
		}
	}
	return nil
}

func FindTopic(app *admin_pb.Platform_App, topicName string) *admin_pb.Platform_App_Topics {
	for _, topics := range app.Topics {
		if topics.Name == topicName {
			return topics
		}
	}
	return nil
}

func CurrentTopicName(platform *admin_pb.Platform, appName string, topicName string) (string, error) {
	app := FindApp(platform, appName)
	if app == nil {
		return "", errors.New(fmt.Sprintf("App '%s' not found", appName))
	}

	topics := FindTopic(app, topicName)
	if topics == nil {
		return "", errors.New(fmt.Sprintf("Topic '%s' not found in App '%s'", topicName, appName))
	}

	pref := buildTopicNamePrefix(platform.Name, app.Name, app.Type)
	return buildTopicName(pref, topics.Name, topics.Current.CreatedAt), nil
}

func min(x, y int) int {
	if x <= y {
		return x
	}
	return y
}

func createMissingTopic(topicName string, topic *admin_pb.Platform_App_Topic, clusterInfos map[string]*clusterInfo) {
	ci, ok := clusterInfos[topic.ClusterName]
	if !ok {
		log.Error().
			Str("ClusterName", topic.ClusterName).
			Msg("Topic with invalid ClusterName")
		return
	}
	if _, c := ci.existingTopics[topicName]; !c {
		err := createTopic(
			ci,
			topicName,
			int(topic.PartitionCount))
		if err != nil {
			log.Error().
				Err(err).
				Str("ClusterName", topic.ClusterName).
				Str("Topic", topicName).
				Msg("Topic creation failure")
			return
		}
	}
}

func createMissingTopics(topicNamePrefix string, topics *admin_pb.Platform_App_Topics, clusterInfos map[string]*clusterInfo) {
	if topics != nil {
		if topics.Current != nil {
			createMissingTopic(
				buildTopicName(topicNamePrefix, topics.Name, topics.Current.CreatedAt),
				topics.Current,
				clusterInfos)
		}
		if topics.Future != nil {
			createMissingTopic(
				buildTopicName(topicNamePrefix, topics.Name, topics.Future.CreatedAt),
				topics.Future,
				clusterInfos)
		}
	}
}

func updateTopics(rtPlat *runtimePlatform) {
	// start admin connections to all clusters
	clusterInfos := make(map[string]*clusterInfo)
	for _, cluster := range rtPlat.platform.Clusters {
		ci, err := NewClusterInfo(cluster)

		if err != nil {
			log.Printf("Unable to connect to cluster '%s', boostrap_servers '%s': %s", cluster.Name, cluster.BootstrapServers, err.Error())
			return
		}

		clusterInfos[cluster.Name] = ci
		defer ci.Close()
		log.Info().
			Str("ClusterName", cluster.Name).
			Str("BootstrapServers", cluster.BootstrapServers).
			Msg("Connected to cluster")
	}

	var appTypesAutoCreate = []string{"GENERAL", "APECS"}

	for _, app := range rtPlat.platform.Apps {
		if contains(appTypesAutoCreate, admin_pb.Platform_App_Type_name[int32(app.Type)]) {
			for _, topics := range app.Topics {
				createMissingTopics(
					buildTopicNamePrefix(rtPlat.platform.Name, app.Name, app.Type),
					topics,
					clusterInfos)
			}
		}
	}
}

func contains(slice []string, item string) bool {
	for _, val := range slice {
		if val == item {
			return true
		}
	}
	return false
}

func buildRuntimeApp(platform *admin_pb.Platform) (*runtimePlatform, error) {
	rtPlat := runtimePlatform{}

	rtPlat.platform = platform

	appJson := protojson.Format(proto.Message(rtPlat.platform))
	sha256Bytes := sha256.Sum256([]byte(appJson))
	rtPlat.hash = hex.EncodeToString(sha256Bytes[:])

	rtPlat.clusters = make(map[string]*admin_pb.Platform_Cluster)
	for idx, cluster := range rtPlat.platform.Clusters {
		if cluster.Name == "" {
			return nil, fmt.Errorf("Cluster %d missing name field", idx)
		}
		if cluster.BootstrapServers == "" {
			return nil, fmt.Errorf("Cluster '%s' missing bootstrap_servers field", cluster.Name)
		}
		// verify clusters only appear once
		if _, ok := rtPlat.clusters[cluster.Name]; ok {
			return nil, fmt.Errorf("Cluster '%s' appears more than once in Platform '%s' definition", cluster.Name, rtPlat.platform.Name)
		}
		rtPlat.clusters[cluster.Name] = cluster
	}

	requiredTopics := map[admin_pb.Platform_App_Type][]string{
		admin_pb.Platform_App_GENERAL: {"error"},
		admin_pb.Platform_App_BATCH:   {"error"},
		admin_pb.Platform_App_APECS:   {"admin", "process", "error", "complete", "storage"},
	}

	rtPlat.apps = make(map[string]*admin_pb.Platform_App)
	for idx, app := range rtPlat.platform.Apps {
		if app.Name == "" {
			return nil, fmt.Errorf("App %d missing name field", idx)
		}

		var topicNames []string
		// validate all topics definitions
		for _, topics := range app.Topics {
			topicNames = append(topicNames, topics.Name)
			if err := validateTopics(topics, rtPlat.clusters); err != nil {
				return nil, fmt.Errorf("App '%s' has invalid '%s' Topics: %s", app.Name, topics.Name, err.Error())
			}
		}

		// validate our expected required topics are there
		for _, req := range requiredTopics[app.Type] {
			if !contains(topicNames, req) {
				return nil, fmt.Errorf("App '%s' missing required '%s' Topics definition", app.Name, req)
			}
		}

		// verify apps only appear once
		if _, ok := rtPlat.apps[app.Name]; ok {
			return nil, fmt.Errorf("App '%s' appears more than once in Platform '%s' definition", app.Name, rtPlat.platform.Name)
		}
		rtPlat.apps[app.Name] = app
	}

	return &rtPlat, nil
}

func validateTopics(topics *admin_pb.Platform_App_Topics, clusters map[string]*admin_pb.Platform_Cluster) error {
	if topics.Name == "" {
		return errors.New("Topics missing Name field")
	}
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
	}
	return nil
}

func validateTopic(topic *admin_pb.Platform_App_Topic, clusters map[string]*admin_pb.Platform_Cluster) error {
	if topic.CreatedAt == 0 {
		return errors.New("Topic missing CreatedAt field")
	}
	if topic.ClusterName == "" {
		return errors.New("Topic missing ClusterName field")
	}
	if _, ok := clusters[topic.ClusterName]; !ok {
		return fmt.Errorf("Topic refers to non-existent cluster: '%s'", topic.ClusterName)
	}
	if topic.PartitionCount < 4 || topic.PartitionCount > 1024 {
		return fmt.Errorf("Topic with out of bounds PartitionCount %d", topic.PartitionCount)
	}
	return nil
}

func manageTopics(ctx context.Context, bootstrapServers string, platformName string) {
	platCh := make(chan admin_pb.Platform, 10)
	go rckafka.ConsumePlatformConfig(ctx, platCh, bootstrapServers, platformName)

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("manageTopics exiting, ctx.Done()")
			return
		case plat := <-platCh:
			rtPlat, err := buildRuntimeApp(&plat)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to buildRuntimeApp")
				continue
			}

			if oldRtPlat == nil || rtPlat.hash != oldRtPlat.hash {
				oldRtPlat = rtPlat
				jsonBytes, _ := protojson.Marshal(proto.Message(rtPlat.platform))
				log.Info().
					Str("Platform", string(jsonBytes)).
					Msg("Platform parsed")

				updateTopics(rtPlat)
			}
		}
	}
}
