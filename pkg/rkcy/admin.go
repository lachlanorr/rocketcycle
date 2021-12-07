// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/version"
)

type ProducerTracker struct {
	platformName  string
	environment   string
	topicProds    map[string]map[string]time.Time
	topicProdsMtx *sync.Mutex
}

func NewProducerTracker(platformName string, environment string) *ProducerTracker {
	pt := &ProducerTracker{
		platformName:  platformName,
		environment:   environment,
		topicProds:    make(map[string]map[string]time.Time),
		topicProdsMtx: &sync.Mutex{},
	}
	return pt
}

func (pt *ProducerTracker) update(pd *rkcypb.ProducerDirective, timestamp time.Time, pingInterval time.Duration) {
	pt.topicProdsMtx.Lock()
	defer pt.topicProdsMtx.Unlock()

	now := time.Now()
	if now.Sub(timestamp) < pingInterval {
		fullTopicName := BuildFullTopicName(
			pt.platformName,
			pt.environment,
			pd.ConcernName,
			pd.ConcernType,
			pd.Topic,
			pd.Generation,
		)
		prodMap, prodMapFound := pt.topicProds[fullTopicName]
		if !prodMapFound {
			prodMap = make(map[string]time.Time)
			pt.topicProds[fullTopicName] = prodMap
		}

		_, prodFound := prodMap[pd.Id]
		if !prodFound {
			log.Info().Msgf("New producer %s:%s", fullTopicName, pd.Id)
		}
		pt.topicProds[fullTopicName][pd.Id] = timestamp
	}
}

func (pt *ProducerTracker) cull(ageLimit time.Duration) {
	pt.topicProdsMtx.Lock()
	defer pt.topicProdsMtx.Unlock()

	now := time.Now()
	for topic, prodMap := range pt.topicProds {
		for id, timestamp := range prodMap {
			age := now.Sub(timestamp)
			if age >= ageLimit {
				log.Info().Msgf("Culling producer %s:%s, age %s", topic, id, age)
				delete(prodMap, id)
			}
		}
		if len(pt.topicProds[topic]) == 0 {
			delete(pt.topicProds, topic)
		}
	}
}

func (pt *ProducerTracker) toTrackedProducers() *rkcypb.TrackedProducers {
	pt.topicProdsMtx.Lock()
	defer pt.topicProdsMtx.Unlock()

	tp := &rkcypb.TrackedProducers{}
	now := time.Now()

	for topic, prodMap := range pt.topicProds {
		for id, timestamp := range prodMap {
			age := now.Sub(timestamp)
			tp.TopicProducers = append(tp.TopicProducers, &rkcypb.TrackedProducers_ProducerInfo{
				Topic:           topic,
				Id:              id,
				TimeSinceUpdate: age.String(),
			})
		}
	}

	return tp
}

func (kplat *KafkaPlatform) cobraAdminServe(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("admin started")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	defer func() {
		signal.Stop(interruptCh)
		cancel()
	}()

	kplat.producerTracker = NewProducerTracker(kplat.name, kplat.environment)

	var wg sync.WaitGroup
	go kplat.managePlatform(ctx, &wg)

	select {
	case <-interruptCh:
		log.Warn().
			Msg("admin stopped")
		cancel()
		wg.Wait()
		return
	}
}

type clusterInfo struct {
	cluster        *rkcypb.Cluster
	admin          *kafka.AdminClient
	existingTopics map[string]struct{}
	brokerCount    int
}

func (ci *clusterInfo) Close() {
	ci.admin.Close()
}

func createTopic(ci *clusterInfo, name string, numPartitions int) error {
	replicationFactor := mini(3, ci.brokerCount)

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
		Str("Cluster", ci.cluster.Name).
		Str("Topic", name).
		Int("NumPartitions", numPartitions).
		Int("ReplicationFactor", replicationFactor).
		Msg("Topic created")

	return nil
}

func newClusterInfo(cluster *rkcypb.Cluster) (*clusterInfo, error) {
	var ci = clusterInfo{}

	config := make(kafka.ConfigMap)
	config.SetKey("bootstrap.servers", cluster.Brokers)

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
		ci.existingTopics[tp.Topic] = gExists
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

func createMissingTopic(topicName string, topic *rkcypb.Concern_Topic, clusterInfos map[string]*clusterInfo) {
	ci, ok := clusterInfos[topic.Cluster]
	if !ok {
		log.Error().
			Str("Cluster", topic.Cluster).
			Msg("Topic with invalid Cluster")
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
				Str("Cluster", topic.Cluster).
				Str("Topic", topicName).
				Msg("Topic creation failure")
			return
		}
	}
}

func createMissingTopics(topicNamePrefix string, topics *rkcypb.Concern_Topics, clusterInfos map[string]*clusterInfo) {
	if topics != nil {
		if topics.Current != nil {
			createMissingTopic(
				BuildTopicName(topicNamePrefix, topics.Name, topics.Current.Generation),
				topics.Current,
				clusterInfos)
		}
		if topics.Future != nil {
			createMissingTopic(
				BuildTopicName(topicNamePrefix, topics.Name, topics.Future.Generation),
				topics.Future,
				clusterInfos)
		}
	}
}

func updateTopics(rtPlatDef *rtPlatformDef) {
	// start admin connections to all clusters
	clusterInfos := make(map[string]*clusterInfo)
	for _, cluster := range rtPlatDef.PlatformDef.Clusters {
		ci, err := newClusterInfo(cluster)

		if err != nil {
			log.Printf("Unable to connect to cluster '%s', brokers '%s': %s", cluster.Name, cluster.Brokers, err.Error())
			return
		}

		clusterInfos[cluster.Name] = ci
		defer ci.Close()
		log.Info().
			Str("Cluster", cluster.Name).
			Str("Brokers", cluster.Brokers).
			Msg("Connected to cluster")
	}

	var concernTypesAutoCreate = []string{"GENERAL", "APECS"}

	for _, concern := range rtPlatDef.PlatformDef.Concerns {
		if contains(concernTypesAutoCreate, rkcypb.Concern_Type_name[int32(concern.Type)]) {
			for _, topics := range concern.Topics {
				createMissingTopics(
					BuildTopicNamePrefix(rtPlatDef.PlatformDef.Name, rtPlatDef.PlatformDef.Environment, concern.Name, concern.Type),
					topics,
					clusterInfos)
			}
		}
	}
}

func (kplat *KafkaPlatform) managePlatform(
	ctx context.Context,
	wg *sync.WaitGroup,
) {
	consumersTopic := ConsumersTopic(kplat.name, kplat.environment)
	adminProdCh := kplat.rawProducer.getProducerCh(ctx, kplat.settings.AdminBrokers, wg)

	platCh := make(chan *PlatformMessage)
	consumePlatformTopic(
		ctx,
		platCh,
		kplat.settings.AdminBrokers,
		kplat.name,
		kplat.environment,
		nil,
		wg,
	)

	prodCh := make(chan *ProducerMessage)
	consumeProducersTopic(
		ctx,
		prodCh,
		kplat.settings.AdminBrokers,
		kplat.name,
		kplat.environment,
		nil,
		wg,
	)

	cullInterval := kplat.AdminPingInterval() * 10
	cullTicker := time.NewTicker(cullInterval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-cullTicker.C:
			// cull stale producers
			kplat.producerTracker.cull(cullInterval)
		case platMsg := <-platCh:
			if (platMsg.Directive & rkcypb.Directive_PLATFORM) != rkcypb.Directive_PLATFORM {
				log.Error().Msgf("Invalid directive for PlatformTopic: %s", platMsg.Directive.String())
				continue
			}

			kplat.currentRtPlatDef = platMsg.NewRtPlatDef

			jsonBytes, _ := protojson.Marshal(proto.Message(kplat.currentRtPlatDef.PlatformDef))
			log.Info().
				Str("PlatformJson", string(jsonBytes)).
				Msg("Platform Replaced")

			platDiff := kplat.currentRtPlatDef.diff(platMsg.OldRtPlatDef, kplat.settings.AdminBrokers, kplat.settings.OtelcolEndpoint)
			updateTopics(kplat.currentRtPlatDef)
			kplat.updateRunner(ctx, adminProdCh, consumersTopic, platDiff)
		case prodMsg := <-prodCh:
			if (prodMsg.Directive & rkcypb.Directive_PRODUCER_STATUS) != rkcypb.Directive_PRODUCER_STATUS {
				log.Error().Msgf("Invalid directive for ProducerTopic: %s", prodMsg.Directive.String())
				continue
			}

			kplat.producerTracker.update(prodMsg.ProducerDirective, prodMsg.Timestamp, kplat.AdminPingInterval()*2)
		}
	}
}

func (kplat *KafkaPlatform) updateRunner(ctx context.Context, adminProdCh ProducerCh, consumersTopic string, platDiff *platformDiff) {
	ctx, span := kplat.telem.StartFunc(ctx)
	defer span.End()
	traceParent := ExtractTraceParent(ctx)

	for _, p := range platDiff.progsToStop {
		msg, err := newKafkaMessage(
			&consumersTopic,
			0,
			&rkcypb.ConsumerDirective{Program: p},
			rkcypb.Directive_CONSUMER_STOP,
			traceParent,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failure in kafkaMessage during updateRunner")
			return
		}

		adminProdCh <- msg
	}

	for _, p := range platDiff.progsToStart {
		msg, err := newKafkaMessage(
			&consumersTopic,
			0,
			&rkcypb.ConsumerDirective{Program: p},
			rkcypb.Directive_CONSUMER_START,
			traceParent,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failure in kafkaMessage during updateRunner")
			return
		}

		adminProdCh <- msg
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
	"service.name":   "rkcy.@platform.@environment.@concern.@system",
	"rkcy.concern":   "@concern",
	"rkcy.system":    "@system",
	"rkcy.topic":     "@topic",
	"rkcy.partition": "@partition",
}

func expandProgs(
	platformName string,
	environment string,
	adminBrokers string,
	otelcolEndpoint string,
	concern *rkcypb.Concern,
	topics *rkcypb.Concern_Topics,
	clusters map[string]*rkcypb.Cluster,
) []*rkcypb.Program {

	substMap := map[string]string{
		"@platform":         platformName,
		"@environment":      environment,
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
