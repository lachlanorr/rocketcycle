// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/version"
)

//go:embed static/admin/docs
var docsFiles embed.FS

func cobraAdminServe(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("admin server started")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go managePlatform(ctx, settings.BootstrapServers, platformName)
	go adminServe(ctx, settings.HttpAddr, settings.GrpcAddr)

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	select {
	case <-interruptCh:
		return
	}
}

func cobraAdminReadPlatform(cmd *cobra.Command, args []string) {
	path := "/v1/platform/read?pretty"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(settings.AdminAddr + path)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to READ")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to ReadAll")
	}

	fmt.Println(string(body))
}

func cobraAdminDecode(cmd *cobra.Command, args []string) {
	path := "/v1/decode"

	slog := log.With().
		Str("Path", path).
		Logger()

	var err error

	intType, err := strconv.Atoi(args[0])
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to parse Type arg")
	}

	buff := Buffer{}
	buff.Type = int32(intType)
	buff.Data, err = base64.StdEncoding.DecodeString(args[1])
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to decode data")
	}

	buffSer, err := protojson.Marshal(&buff)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to marshal Buffer")
	}

	contentRdr := bytes.NewReader(buffSer)
	resp, err := http.Post(settings.AdminAddr+path, "application/json", contentRdr)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to DECODE")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to ReadAll")
	}

	decodeRsp := DecodeResponse{}
	err = protojson.Unmarshal(body, &decodeRsp)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to Unmarshal DecodeResponse")
	}

	fmt.Printf("%s:\n%s\n", decodeRsp.Type, decodeRsp.Json)
}

type adminServer struct {
	UnimplementedAdminServiceServer

	httpAddr string
	grpcAddr string
}

func (srv adminServer) HttpAddr() string {
	return srv.httpAddr
}

func (srv adminServer) GrpcAddr() string {
	return srv.grpcAddr
}

func (adminServer) StaticFiles() http.FileSystem {
	return http.FS(docsFiles)
}

func (adminServer) StaticFilesPathPrefix() string {
	return "/static/admin/docs"
}

func (srv adminServer) RegisterServer(srvReg grpc.ServiceRegistrar) {
	RegisterAdminServiceServer(srvReg, srv)
}

func (adminServer) RegisterHandlerFromEndpoint(
	ctx context.Context,
	mux *runtime.ServeMux,
	endpoint string,
	opts []grpc.DialOption,
) (err error) {
	return RegisterAdminServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

func (adminServer) Platform(ctx context.Context, pa *PlatformArgs) (*Platform, error) {
	if oldRtPlat != nil {
		return oldRtPlat.Platform, nil
	}
	return nil, status.New(codes.FailedPrecondition, "platform not yet initialized").Err()
}

func (adminServer) Decode(ctx context.Context, buffer *Buffer) (*DecodeResponse, error) {
	dec, err := platformImpl.DebugDecoder.Json(buffer)
	if err != nil {
		return nil, err
	}

	typeStr := platformImpl.DebugDecoder.Type(buffer)

	return &DecodeResponse{
		Type: typeStr,
		Json: dec,
	}, nil
}

func adminServe(ctx context.Context, httpAddr string, grpcAddr string) {
	srv := adminServer{httpAddr: httpAddr, grpcAddr: grpcAddr}
	ServeGrpcGateway(ctx, srv)
}

var oldRtPlat *rtPlatform = nil

type clusterInfo struct {
	cluster        *Platform_Cluster
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
		Str("ClusterName", ci.cluster.Name).
		Str("Topic", name).
		Int("NumPartitions", numPartitions).
		Int("ReplicationFactor", replicationFactor).
		Msg("Topic created")

	return nil
}

func newClusterInfo(cluster *Platform_Cluster) (*clusterInfo, error) {
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

func createMissingTopic(topicName string, topic *Platform_Concern_Topic, clusterInfos map[string]*clusterInfo) {
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

func createMissingTopics(topicNamePrefix string, topics *Platform_Concern_Topics, clusterInfos map[string]*clusterInfo) {
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

func updateTopics(rtPlat *rtPlatform) {
	// start admin connections to all clusters
	clusterInfos := make(map[string]*clusterInfo)
	for _, cluster := range rtPlat.Platform.Clusters {
		ci, err := newClusterInfo(cluster)

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

	var concernTypesAutoCreate = []string{"GENERAL", "APECS"}

	for _, concern := range rtPlat.Platform.Concerns {
		if contains(concernTypesAutoCreate, Platform_Concern_Type_name[int32(concern.Type)]) {
			for _, topics := range concern.Topics {
				createMissingTopics(
					BuildTopicNamePrefix(rtPlat.Platform.Name, concern.Name, concern.Type),
					topics,
					clusterInfos)
			}
		}
	}
}

func managePlatform(ctx context.Context, bootstrapServers string, platformName string) {
	adminTopic := AdminTopic(platformName)
	adminProd, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("failed to kafka.NewProducer for admin messages")
	}
	defer adminProd.Close()
	go func() {
		for e := range adminProd.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Error().Msgf("Delivery failed: %v\n", ev.TopicPartition)
				}
			}
		}
	}()

	platCh := make(chan *Platform)
	go consumePlatformConfig(ctx, platCh, bootstrapServers, platformName)

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("managePlatform exiting, ctx.Done()")
			return
		case plat := <-platCh:
			rtPlat, err := newRtPlatform(plat)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to newRtPlatform")
				continue
			}

			if oldRtPlat == nil || rtPlat.Hash != oldRtPlat.Hash {
				jsonBytes, _ := protojson.Marshal(proto.Message(rtPlat.Platform))
				log.Info().
					Str("PlatformJson", string(jsonBytes)).
					Msg("Platform parsed")

				platDiff := rtPlat.diff(oldRtPlat)

				updateTopics(rtPlat)
				updateRunner(adminProd, adminTopic, platDiff)
				oldRtPlat = rtPlat
			}
		}
	}
}

func updateRunner(adminProd *kafka.Producer, adminTopic string, platDiff *platformDiff) {
	for _, p := range platDiff.progsToStop {
		acd := &AdminConsumerDirective{Program: p}
		acdSer, err := proto.Marshal(acd)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal AdminConsumerDirective")
		}
		adminProd.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &adminTopic},
			Value:          acdSer,
			Headers:        standardHeaders(Directive_ADMIN_CONSUMER_STOP, uuid.NewString()),
		}, nil)
	}
	for _, p := range platDiff.progsToStart {
		acd := &AdminConsumerDirective{Program: p}
		acdSer, err := proto.Marshal(acd)
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal AdminConsumerDirective")
		}
		adminProd.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &adminTopic},
			Value:          acdSer,
			Headers:        standardHeaders(Directive_ADMIN_CONSUMER_START, uuid.NewString()),
		}, nil)
	}
}

func substStr(s string, concernName string, clusterBootstrap string, shortTopicName string, fullTopicName string, partition int32) string {
	s = strings.ReplaceAll(s, "@platform", platformName)
	s = strings.ReplaceAll(s, "@bootstrap_servers", clusterBootstrap)
	s = strings.ReplaceAll(s, "@concern", concernName)
	s = strings.ReplaceAll(s, "@system", shortTopicName)
	s = strings.ReplaceAll(s, "@topic", fullTopicName)
	s = strings.ReplaceAll(s, "@partition", strconv.Itoa(int(partition)))
	return s
}

var stdTags map[string]string = map[string]string{
	"service.name":   "rkcy.@platform.@concern.@system",
	"rkcy.concern":   "@concern",
	"rkcy.system":    "@system",
	"rkcy.topic":     "@topic",
	"rkcy.partition": "@partition",
}

func expandProgs(concern *Platform_Concern, topics *Platform_Concern_Topics, clusters map[string]*Platform_Cluster) []*Program {
	progs := make([]*Program, topics.Current.PartitionCount)
	for i := int32(0); i < topics.Current.PartitionCount; i++ {
		topicName := BuildFullTopicName(platformName, concern.Name, concern.Type, topics.Name, topics.Current.Generation)
		cluster := clusters[topics.Current.ClusterName]
		progs[i] = &Program{
			Name:   substStr(topics.ConsumerProgram.Name, concern.Name, cluster.BootstrapServers, topics.Name, topicName, i),
			Args:   make([]string, len(topics.ConsumerProgram.Args)),
			Abbrev: substStr(topics.ConsumerProgram.Abbrev, concern.Name, cluster.BootstrapServers, topics.Name, topicName, i),
			Tags:   make(map[string]string),
		}
		for j := 0; j < len(topics.ConsumerProgram.Args); j++ {
			progs[i].Args[j] = substStr(topics.ConsumerProgram.Args[j], concern.Name, cluster.BootstrapServers, topics.Name, topicName, i)
		}

		for k, v := range stdTags {
			progs[i].Tags[k] = substStr(v, concern.Name, cluster.BootstrapServers, topics.Name, topicName, i)
		}
		if topics.ConsumerProgram.Tags != nil {
			for k, v := range topics.ConsumerProgram.Tags {
				progs[i].Tags[k] = substStr(v, concern.Name, cluster.BootstrapServers, topics.Name, topicName, i)
			}
		}
	}
	return progs
}
