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
	Apps     map[string]*RtApp
	Clusters map[string]*pb.Platform_Cluster
}

type RtApp struct {
	App    *pb.Platform_App
	Topics map[string]*RtTopics
}

type RtTopics struct {
	Topics         *pb.Platform_App_Topics
	CurrentTopic   string
	CurrentCluster *pb.Platform_Cluster
	FutureTopic    string
	FutureCluster  *pb.Platform_Cluster
}

func newRtApp(rtPlatform *RtPlatform, app *pb.Platform_App) (*RtApp, error) {
	rtApp := RtApp{
		App:    app,
		Topics: make(map[string]*RtTopics),
	}
	for _, topics := range app.Topics {
		// verify topics only appear once
		if _, ok := rtApp.Topics[topics.Name]; ok {
			return nil, fmt.Errorf("Topic '%s' appears more than once in App '%s' definition", topics.Name, rtApp.App.Name)
		}
		rtTopics, err := newRtTopics(rtPlatform, &rtApp, topics)
		if err != nil {
			return nil, err
		}
		rtApp.Topics[topics.Name] = rtTopics
	}
	return &rtApp, nil
}

func newRtTopics(rtPlatform *RtPlatform, rtApp *RtApp, topics *pb.Platform_App_Topics) (*RtTopics, error) {
	rtTopics := RtTopics{
		Topics: topics,
	}

	pref := BuildTopicNamePrefix(rtPlatform.Platform.Name, rtApp.App.Name, rtApp.App.Type)
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
		Apps:     make(map[string]*RtApp),
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

	requiredTopics := map[pb.Platform_App_Type][]string{
		pb.Platform_App_GENERAL: {"admin", "error"},
		pb.Platform_App_BATCH:   {"admin", "error"},
		pb.Platform_App_APECS:   {"admin", "process", "error", "complete", "storage"},
	}

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
			if !contains(topicNames, req) {
				return nil, fmt.Errorf("App '%s' missing required '%s' Topics definition", app.Name, req)
			}
		}

		// verify apps only appear once
		if _, ok := rtPlat.Apps[app.Name]; ok {
			return nil, fmt.Errorf("App '%s' appears more than once in Platform '%s' definition", app.Name, rtPlat.Platform.Name)
		}
		rtApp, err := newRtApp(&rtPlat, app)
		if err != nil {
			return nil, err
		}
		rtPlat.Apps[app.Name] = rtApp
	}

	return &rtPlat, nil
}

func validateTopics(topics *pb.Platform_App_Topics, clusters map[string]*pb.Platform_Cluster) error {
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

func validateTopic(topic *pb.Platform_App_Topic, clusters map[string]*pb.Platform_Cluster) error {
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
