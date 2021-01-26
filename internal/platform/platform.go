// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	admin_pb "github.com/lachlanorr/rocketcycle/build/proto/admin"
	"github.com/lachlanorr/rocketcycle/internal/utils"
)

// Platform pb, with some convenience lookup maps
type RtPlatform struct {
	Platform *admin_pb.Platform
	Hash     string
	Apps     map[string]*admin_pb.Platform_App
	Clusters map[string]*admin_pb.Platform_Cluster
}

func NewRtPlatform(platform *admin_pb.Platform) (*RtPlatform, error) {
	rtPlat := RtPlatform{}

	rtPlat.Platform = platform

	platJson := protojson.Format(proto.Message(rtPlat.Platform))
	sha256Bytes := sha256.Sum256([]byte(platJson))
	rtPlat.Hash = hex.EncodeToString(sha256Bytes[:])

	rtPlat.Clusters = make(map[string]*admin_pb.Platform_Cluster)
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

	requiredTopics := map[admin_pb.Platform_App_Type][]string{
		admin_pb.Platform_App_GENERAL: {"error"},
		admin_pb.Platform_App_BATCH:   {"error"},
		admin_pb.Platform_App_APECS:   {"admin", "process", "error", "complete", "storage"},
	}

	rtPlat.Apps = make(map[string]*admin_pb.Platform_App)
	for idx, app := range rtPlat.Platform.Apps {
		if app.Name == "" {
			return nil, fmt.Errorf("App %d missing name field", idx)
		}

		var topicNames []string
		// validate all topics definitions
		for _, topics := range app.Topics {
			topicNames = append(topicNames, topics.Name)
			if err := validateTopics(topics, rtPlat.Clusters); err != nil {
				return nil, fmt.Errorf("App '%s' has invalid '%s' Topics: %s", app.Name, topics.Name, err.Error())
			}
		}

		// validate our expected required topics are there
		for _, req := range requiredTopics[app.Type] {
			if !utils.Contains(topicNames, req) {
				return nil, fmt.Errorf("App '%s' missing required '%s' Topics definition", app.Name, req)
			}
		}

		// verify apps only appear once
		if _, ok := rtPlat.Apps[app.Name]; ok {
			return nil, fmt.Errorf("App '%s' appears more than once in Platform '%s' definition", app.Name, rtPlat.Platform.Name)
		}
		rtPlat.Apps[app.Name] = app
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
