// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type RtPlatformDef struct {
	PlatformDef          *rkcypb.PlatformDef
	Hash                 string
	Concerns             map[string]*RtConcern
	DefaultResponseTopic *RtTopics
	Clusters             map[string]*rkcypb.Cluster
	AdminCluster         *rkcypb.Cluster
	StorageTargets       map[string]*rkcypb.StorageTarget
	PrimaryStorageTarget string
}

type RtConcern struct {
	Concern *rkcypb.Concern
	Topics  map[string]*RtTopics
}

type RtTopics struct {
	Topics                     *rkcypb.Concern_Topics
	CurrentTopic               string
	CurrentTopicPartitionCount int32
	CurrentCluster             *rkcypb.Cluster
	FutureTopic                string
	FutureTopicPartitionCount  int32
	FutureCluster              *rkcypb.Cluster
}

func newRtConcern(rtPlatDef *RtPlatformDef, concern *rkcypb.Concern) (*RtConcern, error) {
	rtConc := RtConcern{
		Concern: concern,
		Topics:  make(map[string]*RtTopics),
	}
	for _, topics := range concern.Topics {
		// verify topics only appear once
		if _, ok := rtConc.Topics[topics.Name]; ok {
			return nil, fmt.Errorf("Topic '%s' appears more than once in Concern '%s' definition", topics.Name, rtConc.Concern.Name)
		}
		rtTops, err := newRtTopics(rtPlatDef, &rtConc, topics)
		if err != nil {
			return nil, err
		}
		rtConc.Topics[topics.Name] = rtTops
	}
	return &rtConc, nil
}

func IsACETopic(topic string) bool {
	return topic == string(ADMIN) || topic == string(ERROR) || topic == string(COMPLETE)
}

func newRtTopics(rtPlatDef *RtPlatformDef, rtConc *RtConcern, topics *rkcypb.Concern_Topics) (*RtTopics, error) {
	rtTops := RtTopics{
		Topics: topics,
	}

	pref := BuildTopicNamePrefix(rtPlatDef.PlatformDef.Name, rtPlatDef.PlatformDef.Environment, rtConc.Concern.Name, rtConc.Concern.Type)
	var ok bool

	rtTops.CurrentTopic = BuildTopicName(pref, topics.Name, topics.Current.Generation)
	rtTops.CurrentTopicPartitionCount = Maxi32(1, topics.Current.PartitionCount)
	rtTops.CurrentCluster, ok = rtPlatDef.Clusters[topics.Current.Cluster]
	if !ok {
		return nil, fmt.Errorf("Topic '%s.%s' has invalid Current Cluster '%s'", rtConc.Concern.Name, topics.Name, topics.Current.Cluster)
	}

	if topics.Future != nil {
		rtTops.FutureTopic = BuildTopicName(pref, topics.Name, topics.Future.Generation)
		rtTops.FutureTopicPartitionCount = Maxi32(1, topics.Future.PartitionCount)
		rtTops.FutureCluster, ok = rtPlatDef.Clusters[topics.Future.Cluster]
		if !ok {
			return nil, fmt.Errorf("Topic '%s.%s' has invalid Future Cluster '%s'", rtConc.Concern.Name, topics.Name, topics.Future.Cluster)
		}
	}

	if IsACETopic(rtTops.Topics.Name) {
		if rtTops.CurrentTopicPartitionCount != 1 {
			return nil, fmt.Errorf("Topic '%s.%s' has invalid partition count current=%d", rtConc.Concern.Name, topics.Name, rtTops.CurrentTopicPartitionCount)
		}
	}

	return &rtTops, nil
}

func initTopic(topic *rkcypb.Concern_Topic, adminCluster *rkcypb.Cluster) *rkcypb.Concern_Topic {
	if topic == nil {
		topic = &rkcypb.Concern_Topic{}
	}

	if topic.Generation <= 0 {
		topic.Generation = 1
	}
	if topic.Cluster == "" {
		topic.Cluster = adminCluster.Name
	}
	if topic.PartitionCount <= 0 {
		topic.PartitionCount = 1
	} else if topic.PartitionCount > MAX_PARTITION {
		topic.PartitionCount = MAX_PARTITION
	}

	return topic
}

func initTopics(
	topics *rkcypb.Concern_Topics,
	adminCluster *rkcypb.Cluster,
	concernType rkcypb.Concern_Type,
	storageTargets []*rkcypb.StorageTarget,
) *rkcypb.Concern_Topics {
	if topics == nil {
		topics = &rkcypb.Concern_Topics{}
	}

	topics.Current = initTopic(topics.Current, adminCluster)
	if topics.Future != nil {
		topics.Future = initTopic(topics.Future, adminCluster)
	}

	if concernType == rkcypb.Concern_APECS {
		topics.ConsumerPrograms = nil
		switch topics.Name {
		case "process":
			prog := &rkcypb.Program{
				Name:   "./@platform",
				Args:   []string{"process", "-t", "@topic", "-p", "@partition", "-e", "@environment", "--otelcol_endpoint", "@otelcol_endpoint", "--admin_brokers", "@admin_brokers", "--consumer_brokers", "@consumer_brokers", "--stream", "@stream"},
				Abbrev: "p/@concern/@partition",
			}
			topics.ConsumerPrograms = append(topics.ConsumerPrograms, prog)
		case "storage":
			for _, stgTgt := range storageTargets {
				if stgTgt.IsPrimary {
					prog := &rkcypb.Program{
						Name: "./@platform",
						Args: []string{"storage", "-t", "@topic", "-p", "@partition", "--storage_target", stgTgt.Name, "-e", "@environment", "--otelcol_endpoint", "@otelcol_endpoint", "--admin_brokers", "@admin_brokers", "--consumer_brokers", "@consumer_brokers", "--stream", "@stream"},
					}
					prog.Abbrev = fmt.Sprintf("s/*%s/@concern/@partition", stgTgt.Name)
					topics.ConsumerPrograms = append(topics.ConsumerPrograms, prog)
				}
			}
		case "storage-scnd":
			for _, stgTgt := range storageTargets {
				if !stgTgt.IsPrimary {
					prog := &rkcypb.Program{
						Name: "./@platform",
						Args: []string{"storage-scnd", "-t", "@topic", "-p", "@partition", "--storage_target", stgTgt.Name, "-e", "@environment", "--otelcol_endpoint", "@otelcol_endpoint", "--admin_brokers", "@admin_brokers", "--consumer_brokers", "@consumer_brokers", "--stream", "@stream"},
					}
					prog.Abbrev = fmt.Sprintf("s/%s/@concern/@partition", stgTgt.Name)
					topics.ConsumerPrograms = append(topics.ConsumerPrograms, prog)
				}
			}
		}
	}

	return topics
}

func NewRtPlatformDef(platDef *rkcypb.PlatformDef, platformName string, environment string) (*RtPlatformDef, error) {
	if platDef.Name != platformName {
		return nil, fmt.Errorf("Platform Name mismatch, '%s' != '%s'", platDef.Name, platformName)
	}
	if platDef.Environment != environment {
		return nil, fmt.Errorf("Environment mismatch, '%s' != '%s'", platDef.Environment, environment)
	}

	rtPlatDef := RtPlatformDef{
		PlatformDef:    platDef,
		Concerns:       make(map[string]*RtConcern),
		Clusters:       make(map[string]*rkcypb.Cluster),
		StorageTargets: make(map[string]*rkcypb.StorageTarget),
	}

	platJson := protojson.Format(proto.Message(rtPlatDef.PlatformDef))
	sha256Bytes := sha256.Sum256([]byte(platJson))
	rtPlatDef.Hash = hex.EncodeToString(sha256Bytes[:])

	if !rtPlatDef.PlatformDef.UpdateTime.IsValid() {
		return nil, fmt.Errorf("Invalid UpdateTime: %s", rtPlatDef.PlatformDef.UpdateTime.AsTime())
	}

	if len(rtPlatDef.PlatformDef.Clusters) <= 0 {
		return nil, fmt.Errorf("No clusters defined")
	}
	for idx, cluster := range rtPlatDef.PlatformDef.Clusters {
		if cluster.Name == "" {
			return nil, fmt.Errorf("Cluster %d missing name field", idx)
		}
		if cluster.Brokers == "" {
			return nil, fmt.Errorf("Cluster '%s' missing brokers field", cluster.Name)
		}
		if cluster.IsAdmin {
			if rtPlatDef.AdminCluster != nil {
				return nil, fmt.Errorf("More than one admin cluster")
			}
			rtPlatDef.AdminCluster = cluster
		}
		// verify clusters only appear once
		if _, ok := rtPlatDef.Clusters[cluster.Name]; ok {
			return nil, fmt.Errorf("Cluster '%s' appears more than once in Platform '%s' definition", cluster.Name, rtPlatDef.PlatformDef.Name)
		}
		rtPlatDef.Clusters[cluster.Name] = cluster
	}
	if rtPlatDef.AdminCluster == nil {
		return nil, fmt.Errorf("No admin cluster defined")
	}

	if len(rtPlatDef.PlatformDef.StorageTargets) <= 0 {
		return nil, fmt.Errorf("No storage targets defined")
	}
	for idx, sttg := range rtPlatDef.PlatformDef.StorageTargets {
		if sttg.Name == "" {
			return nil, fmt.Errorf("Storage target %d missing name field", idx)
		}
		if sttg.Type == "" {
			return nil, fmt.Errorf("Storage target '%s' missing type field", sttg.Name)
		}
		if sttg.IsPrimary {
			if rtPlatDef.PrimaryStorageTarget != "" {
				return nil, fmt.Errorf("More than one primary storage target")
			}
			rtPlatDef.PrimaryStorageTarget = sttg.Name
		}
		// verify clusters only appear once
		if _, ok := rtPlatDef.StorageTargets[sttg.Name]; ok {
			return nil, fmt.Errorf("Storage target '%s' appears more than once in Platform '%s' definition", sttg.Name, rtPlatDef.PlatformDef.Name)
		}
		rtPlatDef.StorageTargets[sttg.Name] = sttg
	}
	if rtPlatDef.PrimaryStorageTarget == "" {
		return nil, fmt.Errorf("No primary storage target defined")
	}

	requiredTopics := map[rkcypb.Concern_Type][]string{
		rkcypb.Concern_GENERAL: {"admin", "error"},
		rkcypb.Concern_BATCH:   {"admin", "error"},
		rkcypb.Concern_APECS:   {"admin", "process", "error", "complete", "storage", "storage-scnd"},
	}

	for idx, concern := range rtPlatDef.PlatformDef.Concerns {
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
			if !Contains(topicNames, req) {
				// conern.Topics will get initialized with reasonable defaults during topic validation below
				concern.Topics = append(concern.Topics, &rkcypb.Concern_Topics{Name: req})
			}
		}

		// ensure APECS concern only has required topics
		if concern.Type == rkcypb.Concern_APECS {
			// simple len check is adequate since we added all required above
			if len(requiredTopics[concern.Type]) != len(concern.Topics) {
				return nil, fmt.Errorf("ApecsConcern %d contains invalid command %+v required vs %+v total", idx, requiredTopics, concern.Topics)
			}
		}

		// validate all topics definitions
		for idx, _ := range concern.Topics {
			concern.Topics[idx] = initTopics(concern.Topics[idx], rtPlatDef.AdminCluster, concern.Type, rtPlatDef.PlatformDef.StorageTargets)
			if err := validateTopics(concern, concern.Topics[idx], rtPlatDef.Clusters); err != nil {
				return nil, fmt.Errorf("Concern '%s' has invalid '%s' Topics: %s", concern.Name, concern.Topics[idx].Name, err.Error())
			}
		}

		// verify concerns only appear once
		if _, ok := rtPlatDef.Concerns[concern.Name]; ok {
			return nil, fmt.Errorf("Concern '%s' appears more than once in Platform '%s' definition", concern.Name, rtPlatDef.PlatformDef.Name)
		}
		rtConc, err := newRtConcern(&rtPlatDef, concern)
		if err != nil {
			return nil, err
		}
		rtPlatDef.Concerns[concern.Name] = rtConc

		// By convention, if there's an "edge" GENERIC concern with a "response" topic, set as default
		if rtConc.Concern.Name == "edge" && rtConc.Concern.Type == rkcypb.Concern_GENERAL {
			respTopic, ok := rtConc.Topics["response"]
			if ok {
				rtPlatDef.DefaultResponseTopic = respTopic
			}
		}
	}

	return &rtPlatDef, nil
}

func NewRtPlatformDefFromJson(platDefJson []byte) (*RtPlatformDef, error) {
	platDef := &rkcypb.PlatformDef{}
	err := protojson.Unmarshal(platDefJson, platDef)
	if err != nil {
		return nil, err
	}

	if !platDef.UpdateTime.IsValid() {
		platDef.UpdateTime = timestamppb.Now()
	}
	return NewRtPlatformDef(platDef, platDef.Name, platDef.Environment)
}

var singlePartitionTopics = map[string]bool{
	"admin":    true,
	"error":    true,
	"complete": true,
}

func validateTopics(concern *rkcypb.Concern, topics *rkcypb.Concern_Topics, clusters map[string]*rkcypb.Cluster) error {
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

func validateTopic(topic *rkcypb.Concern_Topic, clusters map[string]*rkcypb.Cluster) error {
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

type PlatformDiff struct {
	ProgsToStop  []*rkcypb.Program
	ProgsToStart []*rkcypb.Program
}

func (rtPlatDef *RtPlatformDef) getAllProgs(streamType string, adminBrokers string, otelcolEndpoint string) map[string]*rkcypb.Program {
	progs := make(map[string]*rkcypb.Program)
	if rtPlatDef != nil {
		for _, concern := range rtPlatDef.PlatformDef.Concerns {
			for _, topics := range concern.Topics {
				if topics.ConsumerPrograms != nil {
					exProgs := expandProgs(
						rtPlatDef.PlatformDef.Name,
						rtPlatDef.PlatformDef.Environment,
						streamType,
						adminBrokers,
						otelcolEndpoint,
						concern,
						topics,
						rtPlatDef.Clusters,
					)
					for _, p := range exProgs {
						progs[ProgKey(p)] = p
					}
				}
			}
		}
	}
	return progs
}

func (lhs *RtPlatformDef) Diff(rhs *RtPlatformDef, streamType string, adminBrokers string, otelcolEndpoint string) *PlatformDiff {
	d := &PlatformDiff{
		ProgsToStop:  nil,
		ProgsToStart: nil,
	}

	newProgs := lhs.getAllProgs(streamType, adminBrokers, otelcolEndpoint)
	oldProgs := rhs.getAllProgs(streamType, adminBrokers, otelcolEndpoint)

	for k, v := range newProgs {
		if _, ok := oldProgs[k]; !ok {
			d.ProgsToStart = append(d.ProgsToStart, v)
		}
	}

	for k, v := range oldProgs {
		if _, ok := newProgs[k]; !ok {
			d.ProgsToStop = append(d.ProgsToStop, v)
		}
	}

	return d
}

func ProgKey(prog *rkcypb.Program) string {
	// Combine name and args to a string for key lookup
	if prog == nil {
		return "NIL"
	} else {
		return prog.Name + " " + strings.Join(prog.Args, " ")
	}
}

func substStr(
	s string,
	substMap map[string]string,
) string {
	for k, v := range substMap {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}

var gStdTags map[string]string = map[string]string{
	"service.name":     "rkcy.@platform.@environment.@concern.@system",
	"rkcy.environment": "@environment",
	"rkcy.concern":     "@concern",
	"rkcy.system":      "@system",
	"rkcy.topic":       "@topic",
	"rkcy.partition":   "@partition",
}

func expandProgs(
	platformName string,
	environment string,
	streamType string,
	adminBrokers string,
	otelcolEndpoint string,
	concern *rkcypb.Concern,
	topics *rkcypb.Concern_Topics,
	clusters map[string]*rkcypb.Cluster,
) []*rkcypb.Program {

	substMap := map[string]string{
		"@platform":         platformName,
		"@environment":      environment,
		"@stream":           streamType,
		"@admin_brokers":    adminBrokers,
		"@otelcol_endpoint": otelcolEndpoint,
		"@concern":          concern.Name,
	}

	progs := make([]*rkcypb.Program, 0, topics.Current.PartitionCount)
	for _, consProg := range topics.ConsumerPrograms {
		for i := int32(0); i < topics.Current.PartitionCount; i++ {
			substMap["@consumer_brokers"] = clusters[topics.Current.Cluster].Brokers
			substMap["@system"] = topics.Name
			substMap["@topic"] = BuildFullTopicName(platformName, environment, concern.Name, concern.Type, topics.Name, topics.Current.Generation)
			substMap["@partition"] = strconv.Itoa(int(i))
			substMap["@stream"] = streamType

			prog := &rkcypb.Program{
				Name:   substStr(consProg.Name, substMap),
				Args:   make([]string, len(consProg.Args)),
				Abbrev: substStr(consProg.Abbrev, substMap),
				Tags:   make(map[string]string),
			}
			for j := 0; j < len(consProg.Args); j++ {
				prog.Args[j] = substStr(consProg.Args[j], substMap)
			}

			for k, v := range gStdTags {
				prog.Tags[k] = substStr(v, substMap)
			}
			if consProg.Tags != nil {
				for k, v := range consProg.Tags {
					prog.Tags[k] = substStr(v, substMap)
				}
			}
			progs = append(progs, prog)
		}
	}
	return progs
}
