// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rkcy/pkg/rkcy/pb"
	"github.com/lachlanorr/rkcy/version"
)

//go:embed static/admin/docs
var docsFiles embed.FS

func cobraAdminServe(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("rcadmin started")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go manageTopics(ctx, settings.BootstrapServers, platformName)
	go adminServe(ctx, settings.HttpAddr, settings.GrpcAddr)

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	select {
	case <-interruptCh:
		return
	}
}

func cobraAdminGetPlatform(cmd *cobra.Command, args []string) {
	path := "/v1/platform/get?pretty"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(settings.AdminAddr + path)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to GET")
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

type adminServer struct {
	pb.UnimplementedAdminServiceServer

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
	pb.RegisterAdminServiceServer(srvReg, srv)
}

func (adminServer) RegisterHandlerFromEndpoint(
	ctx context.Context,
	mux *runtime.ServeMux,
	endpoint string,
	opts []grpc.DialOption,
) (err error) {
	return pb.RegisterAdminServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

func (adminServer) Platform(ctx context.Context, in *pb.PlatformArgs) (*pb.Platform, error) {
	if oldRtPlat != nil {
		return oldRtPlat.Platform, nil
	}
	return nil, status.New(codes.FailedPrecondition, "platform not yet initialized").Err()
}

func adminServe(ctx context.Context, httpAddr string, grpcAddr string) {
	srv := adminServer{httpAddr: httpAddr, grpcAddr: grpcAddr}
	ServeGrpcGateway(ctx, srv)
}

var oldRtPlat *rtPlatform = nil

type clusterInfo struct {
	cluster        *pb.Platform_Cluster
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

func newClusterInfo(cluster *pb.Platform_Cluster) (*clusterInfo, error) {
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

func createMissingTopic(topicName string, topic *pb.Platform_Concern_Topic, clusterInfos map[string]*clusterInfo) {
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

func createMissingTopics(topicNamePrefix string, topics *pb.Platform_Concern_Topics, clusterInfos map[string]*clusterInfo) {
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
		if contains(concernTypesAutoCreate, pb.Platform_Concern_Type_name[int32(concern.Type)]) {
			for _, topics := range concern.Topics {
				createMissingTopics(
					BuildTopicNamePrefix(rtPlat.Platform.Name, concern.Name, concern.Type),
					topics,
					clusterInfos)
			}
		}
	}
}

func manageTopics(ctx context.Context, bootstrapServers string, platformName string) {
	platCh := make(chan *pb.Platform)
	go consumePlatformConfig(ctx, platCh, bootstrapServers, platformName)

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("manageTopics exiting, ctx.Done()")
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
				oldRtPlat = rtPlat
				jsonBytes, _ := protojson.Marshal(proto.Message(rtPlat.Platform))
				log.Info().
					Str("PlatformJson", string(jsonBytes)).
					Msg("Platform parsed")

				updateTopics(rtPlat)
			}
		}
	}
}
