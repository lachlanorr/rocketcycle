// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type StandardTopicName string

const (
	ADMIN        StandardTopicName = "admin"
	PROCESS                        = "process"
	ERROR                          = "error"
	COMPLETE                       = "complete"
	STORAGE                        = "storage"
	STORAGE_SCND                   = "storage-scnd"
)

var nameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z\-]{1,15}$`)

func IsValidName(name string) bool {
	return nameRe.MatchString(name)
}

func PlatformTopic(platformName string, environment string) string {
	return fmt.Sprintf("%s.%s.%s.platform", RKCY, platformName, environment)
}

func ConfigTopic(platformName string, environment string) string {
	return fmt.Sprintf("%s.%s.%s.config", RKCY, platformName, environment)
}

func ProducersTopic(platformName string, environment string) string {
	return fmt.Sprintf("%s.%s.%s.producers", RKCY, platformName, environment)
}

func ConsumersTopic(platformName string, environment string) string {
	return fmt.Sprintf("%s.%s.%s.consumers", RKCY, platformName, environment)
}

func BuildTopicNamePrefix(platformName string, environment string, concernName string, concernType rkcypb.Concern_Type) string {
	if !IsValidName(platformName) {
		log.Fatal().Msgf("Invalid platformName: %s", platformName)
	}
	if !IsValidName(environment) {
		log.Fatal().Msgf("Invalid environment: %s", environment)
	}
	if !IsValidName(concernName) {
		log.Fatal().Msgf("Invalid concernName: %s", concernName)
	}
	if !IsValidName(concernType.String()) {
		log.Fatal().Msgf("Invalid concernType: %s", concernType.String())
	}
	return fmt.Sprintf("%s.%s.%s.%s.%s", RKCY, platformName, environment, concernName, concernType.String())
}

func BuildTopicName(topicNamePrefix string, name string, generation int32) string {
	if !IsValidName(name) {
		log.Fatal().Msgf("Invalid topicName: %s", name)
	}
	if generation <= 0 {
		log.Fatal().Msgf("Invalid generation: %d", generation)
	}
	return fmt.Sprintf("%s.%s.%04d", topicNamePrefix, name, generation)
}

func BuildFullTopicName(platformName string, environment string, concernName string, concernType rkcypb.Concern_Type, name string, generation int32) string {
	if !IsValidName(platformName) {
		log.Fatal().Msgf("Invalid platformName: %s", platformName)
	}
	if !IsValidName(environment) {
		log.Fatal().Msgf("Invalid environment: %s", environment)
	}
	if !IsValidName(concernName) {
		log.Fatal().Msgf("Invalid concernName: %s", concernName)
	}
	if !IsValidName(concernType.String()) {
		log.Fatal().Msgf("Invalid concernType: %s", concernType.String())
	}
	if !IsValidName(name) {
		log.Fatal().Msgf("Invalid topicName: %s", name)
	}
	if generation <= 0 {
		log.Fatal().Msgf("Invalid generation: %d", generation)
	}
	prefix := BuildTopicNamePrefix(platformName, environment, concernName, concernType)
	return BuildTopicName(prefix, name, generation)
}

type TopicParts struct {
	FullTopic   string
	Platform    string
	Environment string
	Concern     string
	Topic       string
	System      rkcypb.System
	ConcernType rkcypb.Concern_Type
	Generation  int32
}

func ParseFullTopicName(fullTopic string) (*TopicParts, error) {
	parts := strings.Split(fullTopic, ".")
	if len(parts) != 7 || parts[0] != RKCY {
		return nil, fmt.Errorf("Invalid rkcy topic: %s", fullTopic)
	}

	tp := TopicParts{
		FullTopic:   fullTopic,
		Platform:    parts[1],
		Environment: parts[2],
		Concern:     parts[3],
		Topic:       parts[5],
	}

	if !IsValidName(tp.Platform) {
		return nil, fmt.Errorf("Invalid tp.Platform: %s", tp.Platform)
	}
	if !IsValidName(tp.Environment) {
		return nil, fmt.Errorf("Invalid tp.Environment: %s", tp.Environment)
	}
	if !IsValidName(tp.Concern) {
		return nil, fmt.Errorf("Invalid tp.Concern: %s", tp.Concern)
	}
	if !IsValidName(tp.Topic) {
		return nil, fmt.Errorf("Invalid tp.Topic: %s", tp.Topic)
	}

	if tp.Topic == PROCESS {
		tp.System = rkcypb.System_PROCESS
	} else if tp.Topic == STORAGE {
		tp.System = rkcypb.System_STORAGE
	} else if tp.Topic == STORAGE_SCND {
		tp.System = rkcypb.System_STORAGE_SCND
	} else {
		tp.System = rkcypb.System_NO_SYSTEM
	}

	concernType, ok := rkcypb.Concern_Type_value[parts[4]]
	if !ok {
		return nil, fmt.Errorf("Invalid rkcy topic, unable to parse ConcernType: %s", fullTopic)
	}
	tp.ConcernType = rkcypb.Concern_Type(concernType)

	generation, err := strconv.Atoi(parts[6])
	if err != nil {
		return nil, fmt.Errorf("Invalid rkcy topic, unable to parse Generation: %s", fullTopic)
	}
	tp.Generation = int32(generation)
	if tp.Generation < 0 {
		return nil, fmt.Errorf("Invalid tp.Generation: %d", tp.Generation)
	}

	return &tp, nil
}

func CreatePlatformTopics(
	ctx context.Context,
	strmprov StreamProvider,
	platform string,
	environment string,
	adminBrokers string,
) error {
	topicNames := []string{
		PlatformTopic(platform, environment),
		ConfigTopic(platform, environment),
		ProducersTopic(platform, environment),
		ConsumersTopic(platform, environment),
	}

	// connect to kafka and make sure we have our platform topic
	admin, err := strmprov.NewAdminClient(adminBrokers)
	if err != nil {
		return err
	}

	md, err := admin.GetMetadata(nil, true, 1000)
	if err != nil {
		return errors.New("Failed to GetMetadata")
	}

	for _, topicName := range topicNames {
		_, ok := md.Topics[topicName]
		if !ok { // platform topic doesn't exist
			result, err := admin.CreateTopics(
				ctx,
				[]kafka.TopicSpecification{
					{
						Topic:             topicName,
						NumPartitions:     1,
						ReplicationFactor: Mini(3, len(md.Brokers)),
						Config: map[string]string{
							"retention.ms":    "-1",
							"retention.bytes": strconv.Itoa(int(PLATFORM_TOPIC_RETENTION_BYTES)),
						},
					},
				},
				nil,
			)
			if err != nil {
				return fmt.Errorf("Failed to create platform topic: %s", topicName)
			}
			for _, res := range result {
				if res.Error.Code() != kafka.ErrNoError {
					return fmt.Errorf("Failed to create platform topic: %s", topicName)
				}
			}
			log.Info().
				Str("Topic", topicName).
				Msg("Platform topic created")
		}
	}
	return nil
}

func UpdateTopics(
	ctx context.Context,
	strmprov StreamProvider,
	platDef *rkcypb.PlatformDef,
) error {
	// start admin connections to all clusters
	clusterInfos := make(map[string]*clusterInfo)
	for _, cluster := range platDef.Clusters {
		ci, err := newClusterInfo(strmprov, cluster)

		if err != nil {
			log.Printf("Unable to connect to cluster '%s', brokers '%s': %s", cluster.Name, cluster.Brokers, err.Error())
			return err
		}

		clusterInfos[cluster.Name] = ci
		defer ci.Close()
		log.Info().
			Str("Cluster", cluster.Name).
			Str("Brokers", cluster.Brokers).
			Msg("Connected to cluster")
	}

	var concernTypesAutoCreate = []string{"GENERAL", "APECS"}

	for _, concern := range platDef.Concerns {
		if Contains(concernTypesAutoCreate, rkcypb.Concern_Type_name[int32(concern.Type)]) {
			for _, topics := range concern.Topics {
				createMissingTopics(
					ctx,
					BuildTopicNamePrefix(platDef.Name, platDef.Environment, concern.Name, concern.Type),
					topics,
					clusterInfos,
				)
			}
		}
	}
	return nil
}

type clusterInfo struct {
	cluster        *rkcypb.Cluster
	admin          AdminClient
	existingTopics map[string]bool
	brokerCount    int
}

func (ci *clusterInfo) Close() {
	ci.admin.Close()
}

func createTopic(
	ctx context.Context,
	ci *clusterInfo,
	name string,
	numPartitions int,
) error {
	replicationFactor := Mini(3, ci.brokerCount)

	topicSpec := []kafka.TopicSpecification{
		{
			Topic:             name,
			NumPartitions:     numPartitions,
			ReplicationFactor: replicationFactor,
		},
	}

	timeout, _ := time.ParseDuration("30s")
	ctx, cancel := context.WithTimeout(ctx, timeout)
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
		Str("Cluster", ci.cluster.Name).
		Str("Topic", name).
		Int("NumPartitions", numPartitions).
		Int("ReplicationFactor", replicationFactor).
		Msg("Topic created")

	return nil
}

func newClusterInfo(
	strmprov StreamProvider,
	cluster *rkcypb.Cluster,
) (*clusterInfo, error) {
	var ci = clusterInfo{}

	var err error
	ci.admin, err = strmprov.NewAdminClient(cluster.Brokers)
	if err != nil {
		return nil, err
	}
	ci.cluster = cluster

	ci.existingTopics = make(map[string]bool)

	md, err := ci.admin.GetMetadata(nil, true, 1000)

	if err != nil {
		defer ci.admin.Close()
		return nil, err
	}

	sortedTopics := make([]string, 0, len(md.Topics))
	ci.brokerCount = len(md.Brokers)
	for _, tp := range md.Topics {
		sortedTopics = append(sortedTopics, tp.Topic)
		ci.existingTopics[tp.Topic] = true
	}

	sort.Strings(sortedTopics)
	for _, topicName := range sortedTopics {
		log.Info().
			Str("Cluster", cluster.Name).
			Str("Topic", topicName).
			Msg("Topic found")
	}

	return &ci, nil
}

func createMissingTopic(
	ctx context.Context,
	topicName string,
	topic *rkcypb.Concern_Topic,
	clusterInfos map[string]*clusterInfo,
) {
	ci, ok := clusterInfos[topic.Cluster]
	if !ok {
		log.Error().
			Str("Cluster", topic.Cluster).
			Msg("Topic with invalid Cluster")
		return
	}
	if !ci.existingTopics[topicName] {
		err := createTopic(
			ctx,
			ci,
			topicName,
			int(topic.PartitionCount))
		if err != nil {
			log.Error().
				Err(err).
				Str("Cluster", topic.Cluster).
				Str("Topic", topicName).
				Msg("Topic creation failure")
			return
		}
	}
}

func createMissingTopics(
	ctx context.Context,
	topicNamePrefix string,
	topics *rkcypb.Concern_Topics,
	clusterInfos map[string]*clusterInfo,
) {
	if topics != nil {
		if topics.Current != nil {
			createMissingTopic(
				ctx,
				BuildTopicName(topicNamePrefix, topics.Name, topics.Current.Generation),
				topics.Current,
				clusterInfos,
			)
		}
		if topics.Future != nil {
			createMissingTopic(
				ctx,
				BuildTopicName(topicNamePrefix, topics.Name, topics.Future.Generation),
				topics.Future,
				clusterInfos,
			)
		}
	}
}
