// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rkcy/pkg/rkcy/pb"
)

// Platform pb, with some convenience lookup maps
type RtPlatform struct {
	Platform *pb.Platform
	Hash     string
	Concerns map[string]*RtConcern
	Clusters map[string]*pb.Platform_Cluster
}

type RtConcern struct {
	Concern *pb.Platform_Concern
	Topics  map[string]*RtTopics
}

type RtTopics struct {
	Topics         *pb.Platform_Concern_Topics
	CurrentTopic   string
	CurrentCluster *pb.Platform_Cluster
	FutureTopic    string
	FutureCluster  *pb.Platform_Cluster
}

func newRtConcern(rtPlatform *RtPlatform, concern *pb.Platform_Concern) (*RtConcern, error) {
	rtConcern := RtConcern{
		Concern: concern,
		Topics:  make(map[string]*RtTopics),
	}
	for _, topics := range concern.Topics {
		// verify topics only appear once
		if _, ok := rtConcern.Topics[topics.Name]; ok {
			return nil, fmt.Errorf("Topic '%s' appears more than once in Concern '%s' definition", topics.Name, rtConcern.Concern.Name)
		}
		rtTopics, err := newRtTopics(rtPlatform, &rtConcern, topics)
		if err != nil {
			return nil, err
		}
		rtConcern.Topics[topics.Name] = rtTopics
	}
	return &rtConcern, nil
}

func newRtTopics(rtPlatform *RtPlatform, rtConcern *RtConcern, topics *pb.Platform_Concern_Topics) (*RtTopics, error) {
	rtTopics := RtTopics{
		Topics: topics,
	}

	pref := BuildTopicNamePrefix(rtPlatform.Platform.Name, rtConcern.Concern.Name, rtConcern.Concern.Type)
	var ok bool

	rtTopics.CurrentTopic = BuildTopicName(pref, topics.Name, topics.Current.Generation)
	rtTopics.CurrentCluster, ok = rtPlatform.Clusters[topics.Current.ClusterName]
	if !ok {
		return nil, fmt.Errorf("Topic '%s' has invalid Current ClusterName '%s'", topics.Name, topics.Current.ClusterName)
	}

	if topics.Future != nil {
		rtTopics.FutureTopic = BuildTopicName(pref, topics.Name, topics.Future.Generation)
		rtTopics.FutureCluster, ok = rtPlatform.Clusters[topics.Future.ClusterName]
		if !ok {
			return nil, fmt.Errorf("Topic '%s' has invalid Future ClusterName '%s'", topics.Name, topics.Future.ClusterName)
		}
	}
	return &rtTopics, nil
}

func NewRtPlatform(platform *pb.Platform) (*RtPlatform, error) {
	rtPlat := RtPlatform{
		Platform: platform,
		Concerns: make(map[string]*RtConcern),
		Clusters: make(map[string]*pb.Platform_Cluster),
	}

	platJson := protojson.Format(proto.Message(rtPlat.Platform))
	sha256Bytes := sha256.Sum256([]byte(platJson))
	rtPlat.Hash = hex.EncodeToString(sha256Bytes[:])

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

		var topicNames []string
		// validate all topics definitions
		for _, topics := range concern.Topics {
			topicNames = append(topicNames, topics.Name)
			if err := validateTopics(topics, rtPlat.Clusters); err != nil {
				return nil, fmt.Errorf("Concern '%s' has invalid '%s' Topics: %s", concern.Name, topics.Name, err.Error())
			}
		}

		// validate our expected required topics are there
		for _, req := range requiredTopics[concern.Type] {
			if !contains(topicNames, req) {
				return nil, fmt.Errorf("Concern '%s' missing required '%s' Topics definition", concern.Name, req)
			}
		}

		// verify concerns only appear once
		if _, ok := rtPlat.Concerns[concern.Name]; ok {
			return nil, fmt.Errorf("Concern '%s' appears more than once in Platform '%s' definition", concern.Name, rtPlat.Platform.Name)
		}
		rtConcern, err := newRtConcern(&rtPlat, concern)
		if err != nil {
			return nil, err
		}
		rtPlat.Concerns[concern.Name] = rtConcern
	}

	return &rtPlat, nil
}

func validateTopics(topics *pb.Platform_Concern_Topics, clusters map[string]*pb.Platform_Cluster) error {
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
