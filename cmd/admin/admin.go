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
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/gaeneco/pb"
	"github.com/lachlanorr/gaeneco/version"
)

// Metadata pb, with some convenience lookup maps
type runtimeMeta struct {
	meta     *pb.Metadata
	hash     string
	apps     map[string]*pb.Metadata_App
	clusters map[string]*pb.Metadata_Cluster
}

var exists struct{}

type clusterInfo struct {
	cluster        *pb.Metadata_Cluster
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
		Str("TopicName", name).
		Int("NumPartitions", numPartitions).
		Int("ReplicationFactor", replicationFactor).
		Msg("Topic created")

	return nil
}

func NewClusterInfo(cluster *pb.Metadata_Cluster) (*clusterInfo, error) {
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
			Str("TopicName", topicName).
			Msg("Topic found")
	}

	return &ci, nil
}

func topicNamePrefix(metaName string, appName string, appType pb.Metadata_App_Type) string {
	return fmt.Sprintf("gaeneco.%s.%s.%s", metaName, appName, pb.Metadata_App_Type_name[int32(appType)])
}

func topicName(topicNamePrefix string, name string, createdAt int64) string {
	return fmt.Sprintf("%s.%s.%010d", topicNamePrefix, name, createdAt)
}

func min(x, y int) int {
	if x <= y {
		return x
	}
	return y
}

func createMissingTopic(topicName string, topic *pb.Metadata_App_Topic, clusterInfos map[string]*clusterInfo) {
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
				Str("ClusterName", topic.ClusterName).
				Str("TopicName", topicName).
				Str("Error", err.Error()).
				Msg("Topic creation failure")
			return
		}
	}
}

func createMissingTopics(topicNamePrefix string, topics *pb.Metadata_App_Topics, clusterInfos map[string]*clusterInfo) {
	if topics != nil {
		if topics.Current != nil {
			createMissingTopic(
				topicName(topicNamePrefix, topics.Name, topics.Current.CreatedAt),
				topics.Current,
				clusterInfos)
		}
		if topics.Future != nil {
			createMissingTopic(
				topicName(topicNamePrefix, topics.Name, topics.Future.CreatedAt),
				topics.Future,
				clusterInfos)
		}
	}
}

func updateTopics(rtmeta *runtimeMeta) {
	// start admin connections to all clusters
	clusterInfos := make(map[string]*clusterInfo)
	for _, cluster := range rtmeta.meta.Clusters {
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

	for _, app := range rtmeta.meta.Apps {
		for _, topics := range app.Topics {
			createMissingTopics(
				topicNamePrefix(rtmeta.meta.Name, app.Name, app.Type),
				topics,
				clusterInfos)
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

func buildRuntimeApp(meta *pb.Metadata) (*runtimeMeta, error) {
	rtmeta := runtimeMeta{}

	rtmeta.meta = meta

	appJson := protojson.Format(proto.Message(rtmeta.meta))
	sha256Bytes := sha256.Sum256([]byte(appJson))
	rtmeta.hash = hex.EncodeToString(sha256Bytes[:])

	rtmeta.clusters = make(map[string]*pb.Metadata_Cluster)
	for idx, cluster := range rtmeta.meta.Clusters {
		if cluster.Name == "" {
			return nil, fmt.Errorf("Cluster %d missing name field", idx)
		}
		if cluster.BootstrapServers == "" {
			return nil, fmt.Errorf("Cluster '%s' missing bootstrap_servers field", cluster.Name)
		}
		// verify clusters only appear once
		if _, ok := rtmeta.clusters[cluster.Name]; ok {
			return nil, fmt.Errorf("Cluster '%s' appears more than once in Metadata '%s' definition", cluster.Name, rtmeta.meta.Name)
		}
		rtmeta.clusters[cluster.Name] = cluster
	}

	requiredTopics := map[pb.Metadata_App_Type][]string{
		pb.Metadata_App_GENERAL: {"error"},
		pb.Metadata_App_BATCH:   {"error"},
		pb.Metadata_App_APECS:   {"admin", "process", "error", "complete", "storage"},
	}

	rtmeta.apps = make(map[string]*pb.Metadata_App)
	for idx, app := range rtmeta.meta.Apps {
		if app.Name == "" {
			return nil, fmt.Errorf("App %d missing name field", idx)
		}

		var topicNames []string
		// validate all topics definitions
		for _, topics := range app.Topics {
			topicNames = append(topicNames, topics.Name)
			if err := validateTopics(topics, rtmeta.clusters); err != nil {
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
		if _, ok := rtmeta.apps[app.Name]; ok {
			return nil, fmt.Errorf("App '%s' appears more than once in Metadata '%s' definition", app.Name, rtmeta.meta.Name)
		}
		rtmeta.apps[app.Name] = app
	}

	return &rtmeta, nil
}

func validateTopics(topics *pb.Metadata_App_Topics, clusters map[string]*pb.Metadata_Cluster) error {
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

func validateTopic(topic *pb.Metadata_App_Topic, clusters map[string]*pb.Metadata_Cluster) error {
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

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("admin started")

	configPath := "./metadata.json"

	var oldRtmeta *runtimeMeta

	for {
		time.Sleep(time.Second)

		appJsonBytes, err := ioutil.ReadFile(configPath)
		if err != nil {
			log.Error().
				Str("Error", err.Error()).
				Str("Path", configPath).
				Msg("Error reading config file")
			continue
		}

		app := pb.Metadata{}
		err = protojson.Unmarshal(appJsonBytes, proto.Message(&app))
		if err != nil {
			log.Error().
				Str("Error", err.Error()).
				Str("Path", configPath).
				Str("Metadata", string(appJsonBytes)).
				Msg("Error unmarshalling config file")
			continue
		}

		rtmeta, err := buildRuntimeApp(&app)
		if err != nil {
			log.Error().
				Str("Error", err.Error()).
				Str("Path", configPath).
				Str("Metadata", string(appJsonBytes)).
				Msg("Error building runtimeMeta")
			continue
		}

		if oldRtmeta == nil || rtmeta.hash != oldRtmeta.hash {
			oldRtmeta = rtmeta
			jsonBytes, _ := protojson.Marshal(proto.Message(rtmeta.meta))
			log.Info().
				Str("Metadata", string(jsonBytes)).
				Msg("Metadata parsed")

			updateTopics(rtmeta)
		}
	}

}
