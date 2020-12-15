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
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/gaeneco/pb"
)

// Application pb, with some convenience lookup maps
type runtimeApp struct {
	app      *pb.Application
	hash     string
	models   map[string]*pb.Application_Model
	clusters map[string]*pb.Application_Cluster
}

func createTopic(admin *kafka.AdminClient, name string, numPartitions int, replicationFactor int) error {
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

	result, err := admin.CreateTopics(ctx, topicSpec, nil)
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
		Str("TopicName", name).
		Int("NumPartitions", numPartitions).
		Int("ReplicationFactor", replicationFactor).
		Msg("Topic created")

	return nil
}

var exists struct{}

type clusterInfo struct {
	admin          *kafka.AdminClient
	existingTopics map[string]struct{}
}

func (ci *clusterInfo) Close() {
	ci.admin.Close()
}

func NewClusterInfo(cluster *pb.Application_Cluster) (*clusterInfo, error) {
	var ci = clusterInfo{}

	config := make(kafka.ConfigMap)
	config.SetKey("bootstrap.servers", cluster.BootstrapServers)

	var err error
	ci.admin, err = kafka.NewAdminClient(&config)
	if err != nil {
		return nil, err
	}

	ci.existingTopics = make(map[string]struct{})

	md, err := ci.admin.GetMetadata(nil, true, 1000)

	if err != nil {
		defer ci.admin.Close()
		return nil, err
	}

	for _, tp := range md.Topics {
		log.Info().
			Str("TopicName", tp.Topic).
			Msg("Topic found")
		ci.existingTopics[tp.Topic] = exists
	}

	return &ci, nil
}

func createMissingTopic(clusterInfos map[string]*clusterInfo, topic *pb.Application_Model_Topic) {
	ci, ok := clusterInfos[topic.ClusterName]
	if !ok {
		log.Error().
			Str("ClusterName", topic.ClusterName).
			Msg("Topic with invalid ClusterName")
		return
	}
	if _, c := ci.existingTopics[topic.Name]; !c {
		err := createTopic(ci.admin, topic.Name, int(topic.PartitionCount), 1)
		if err != nil {
			log.Error().
				Str("ClusterName", topic.ClusterName).
				Str("TopicName", topic.Name).
				Str("Error", err.Error()).
				Msg("Topic creation failure")
			return
		}
	}
}

func createMissingTopics(clusterInfos map[string]*clusterInfo, topics *pb.Application_Model_Topics) {
	if topics != nil {
		if topics.Current != nil {
			createMissingTopic(clusterInfos, topics.Current)
		}
		if topics.Future != nil {
			createMissingTopic(clusterInfos, topics.Future)
		}
	}
}

func updateTopics(rtapp *runtimeApp) {
	// start admin connections to all clusters
	clusterInfos := make(map[string]*clusterInfo)
	for _, cluster := range rtapp.app.Clusters {
		ci, err := NewClusterInfo(cluster)

		if err != nil {
			log.Printf("Unable to connect to cluster '%s', boostrap_servers '%s': %s", cluster.Name, cluster.BootstrapServers, err.Error())
		}

		clusterInfos[cluster.Name] = ci
		defer ci.Close()
		log.Info().
			Str("ClusterName", cluster.Name).
			Str("BootstrapServers", cluster.BootstrapServers).
			Msg("Connected to cluster")
	}

	for _, model := range rtapp.app.Models {
		for _, topics := range model.Topics {
			createMissingTopics(clusterInfos, topics)
		}
	}
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	configPath := "./application.json"

	var oldRtApp *runtimeApp

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

		app := pb.Application{}
		err = protojson.Unmarshal(appJsonBytes, proto.Message(&app))
		if err != nil {
			log.Error().
				Str("Error", err.Error()).
				Str("Path", configPath).
				Str("Content", string(appJsonBytes)).
				Msg("Error unmarshalling config file")
			continue
		}

		rtapp, err := buildRuntimeApp(&app)
		if err != nil {
			log.Error().
				Str("Error", err.Error()).
				Str("Path", configPath).
				Str("Content", string(appJsonBytes)).
				Msg("Error building runtimeApp")
			continue
		}

		if oldRtApp == nil || rtapp.hash != oldRtApp.hash {
			oldRtApp = rtapp
			configStr := protojson.Format(proto.Message(rtapp.app))
			log.Info().
				Msg(configStr)

			updateTopics(rtapp)
		}
	}

}

func buildRuntimeApp(app *pb.Application) (*runtimeApp, error) {
	rtapp := runtimeApp{}

	rtapp.app = app

	appJson := protojson.Format(proto.Message(rtapp.app))
	sha256Bytes := sha256.Sum256([]byte(appJson))
	rtapp.hash = hex.EncodeToString(sha256Bytes[:])

	rtapp.clusters = make(map[string]*pb.Application_Cluster)
	for idx, cluster := range rtapp.app.Clusters {
		if cluster.Name == "" {
			return nil, fmt.Errorf("Cluster %d missing name field", idx)
		}
		if cluster.BootstrapServers == "" {
			return nil, fmt.Errorf("Cluster '%s' missing bootstrap_servers field", cluster.Name)
		}
		// verify clusters only appear once
		if _, ok := rtapp.clusters[cluster.Name]; ok {
			return nil, fmt.Errorf("Cluster '%s' appears more than once in Application '%s' definition", cluster.Name, rtapp.app.Name)
		}
		rtapp.clusters[cluster.Name] = cluster
	}

	rtapp.models = make(map[string]*pb.Application_Model)
	for idx, model := range rtapp.app.Models {
		if model.Name == "" {
			return nil, fmt.Errorf("Model %d missing name field", idx)
		}

		// validate our expected required topics are there
		requiredTopics := []string{"control", "process", "error"}
		for _, req := range requiredTopics {
			if _, ok := model.Topics[req]; !ok {
				return nil, fmt.Errorf("Model '%s' missing required '%s' Topics definition", model.Name, req)
			}
		}
		// validate all topics definitions
		for name, topics := range model.Topics {
			if err := validateTopics(topics, rtapp.clusters); err != nil {
				return nil, fmt.Errorf("Model '%s' has invalid '%s' Topics: %s", model.Name, name, err.Error())
			}
		}

		// verify models only appear once
		if _, ok := rtapp.models[model.Name]; ok {
			return nil, fmt.Errorf("Model '%s' appears more than once in Application '%s' definition", model.Name, rtapp.app.Name)
		}
		rtapp.models[model.Name] = model
	}

	return &rtapp, nil
}

func validateTopics(topics *pb.Application_Model_Topics, clusters map[string]*pb.Application_Cluster) error {
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

func validateTopic(topic *pb.Application_Model_Topic, clusters map[string]*pb.Application_Cluster) error {
	if topic.Name == "" {
		return errors.New("Topic missing Name field")
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
