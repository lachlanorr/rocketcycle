// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
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
	"github.com/lachlanorr/rocketcycle/internal/rkcy"
)

var exists struct{}
var oldRtPlat *rkcy.RtPlatform = nil

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

func buildTopicName(topicNamePrefix string, name string, generation int32) string {
	return fmt.Sprintf("%s.%s.%04d", topicNamePrefix, name, generation)
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
	return buildTopicName(pref, topics.Name, topics.Current.Generation), nil
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
				buildTopicName(topicNamePrefix, topics.Name, topics.Current.Generation),
				topics.Current,
				clusterInfos)
		}
		if topics.Future != nil {
			createMissingTopic(
				buildTopicName(topicNamePrefix, topics.Name, topics.Future.Generation),
				topics.Future,
				clusterInfos)
		}
	}
}

func updateTopics(rtPlat *rkcy.RtPlatform) {
	// start admin connections to all clusters
	clusterInfos := make(map[string]*clusterInfo)
	for _, cluster := range rtPlat.Platform.Clusters {
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

	for _, app := range rtPlat.Platform.Apps {
		if rkcy.Contains(appTypesAutoCreate, admin_pb.Platform_App_Type_name[int32(app.Type)]) {
			for _, topics := range app.Topics {
				createMissingTopics(
					buildTopicNamePrefix(rtPlat.Platform.Name, app.Name, app.Type),
					topics,
					clusterInfos)
			}
		}
	}
}

func manageTopics(ctx context.Context, bootstrapServers string, platformName string) {
	platCh := make(chan admin_pb.Platform)
	go rkcy.ConsumePlatformConfig(ctx, platCh, bootstrapServers, platformName)

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("manageTopics exiting, ctx.Done()")
			return
		case plat := <-platCh:
			rtPlat, err := rkcy.NewRtPlatform(&plat)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to NewRtPlatform")
				continue
			}

			if oldRtPlat == nil || rtPlat.Hash != oldRtPlat.Hash {
				oldRtPlat = rtPlat
				jsonBytes, _ := protojson.Marshal(proto.Message(rtPlat.Platform))
				log.Info().
					Str("Platform", string(jsonBytes)).
					Msg("Platform parsed")

				updateTopics(rtPlat)
			}
		}
	}
}
