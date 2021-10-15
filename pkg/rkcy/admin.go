// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
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

	"github.com/lachlanorr/rocketcycle/version"
)

//go:embed static/admin/docs
var gDocsFiles embed.FS

var gAdminPingInterval = 1 * time.Second
var gPlatformRepublishInterval = 60 * time.Second

var gCurrentRtPlat *rtPlatform = nil

type ProducerTracker struct {
	platformName  string
	topicProds    map[string]map[string]time.Time
	topicProdsMtx *sync.Mutex
}

var gProducerTracker *ProducerTracker

func NewProducerTracker(platformName string) *ProducerTracker {
	pt := &ProducerTracker{
		platformName:  platformName,
		topicProds:    make(map[string]map[string]time.Time),
		topicProdsMtx: &sync.Mutex{},
	}
	return pt
}

func (pt *ProducerTracker) update(pd *ProducerDirective, timestamp time.Time, pingInterval time.Duration) {
	pt.topicProdsMtx.Lock()
	defer pt.topicProdsMtx.Unlock()

	now := time.Now()
	if now.Sub(timestamp) < pingInterval {
		fullTopicName := BuildFullTopicName(
			pt.platformName,
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

func (pt *ProducerTracker) toTrackedProducers() *TrackedProducers {
	pt.topicProdsMtx.Lock()
	defer pt.topicProdsMtx.Unlock()

	tp := &TrackedProducers{}
	now := time.Now()

	for topic, prodMap := range pt.topicProds {
		for id, timestamp := range prodMap {
			age := now.Sub(timestamp)
			tp.TopicProducers = append(tp.TopicProducers, &TrackedProducers_ProducerInfo{
				Topic:           topic,
				Id:              id,
				TimeSinceUpdate: age.String(),
			})
		}
	}

	return tp
}

func cobraAdminServe(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("admin server started")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	defer func() {
		signal.Stop(interruptCh)
		cancel()
	}()

	gProducerTracker = NewProducerTracker(gPlatformName)

	var wg sync.WaitGroup
	wg.Add(1)
	go managePlatform(ctx, gPlatformName, &wg)
	go adminServe(ctx, gSettings.HttpAddr, gSettings.GrpcAddr, &wg)

	select {
	case <-interruptCh:
		log.Warn().
			Msg("Admin server stopped")
		cancel()
		wg.Wait()
		return
	}
}

func cobraAdminReadPlatform(cmd *cobra.Command, args []string) {
	path := "/v1/platform/read?pretty"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(gSettings.AdminAddr + path)
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

func cobraAdminReadConfig(cmd *cobra.Command, args []string) {
	path := "/v1/config/read"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(gSettings.AdminAddr + path)
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
		return
	}

	confRsp := &ConfigReadResponse{}
	err = protojson.Unmarshal(body, confRsp)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to Unmarshal ConfigResponse")
		return
	}

	var prettyJson bytes.Buffer
	err = json.Indent(&prettyJson, []byte(confRsp.ConfigJson), "", "  ")
	if err != nil {
		slog.Fatal().
			Err(err).
			Str("Json", confRsp.ConfigJson).
			Msg("Failed to prettify json")
		return
	}

	fmt.Printf("%s\n", string(prettyJson.Bytes()))
}

func cobraAdminReadProducers(cmd *cobra.Command, args []string) {
	path := "/v1/producers/read?pretty"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(gSettings.AdminAddr + path)
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

func cobraAdminDecodeInstance(cmd *cobra.Command, args []string) {
	path := "/v1/decode/instance"
	slog := log.With().
		Str("Path", path).
		Logger()

	var err error

	rpcArgs := DecodeInstanceArgs{
		Concern:   args[0],
		Payload64: args[1],
	}

	rpcArgsSer, err := protojson.Marshal(&rpcArgs)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to marshal rpcArgs")
	}

	contentRdr := bytes.NewReader(rpcArgsSer)
	resp, err := http.Post(gSettings.AdminAddr+path, "application/json", contentRdr)
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

	fmt.Printf("%s\n", decodeRsp.Json)
}

type adminServer struct {
	UnimplementedAdminServiceServer

	httpAddr string
	grpcAddr string

	confMgr *ConfigMgr
}

func (srv adminServer) HttpAddr() string {
	return srv.httpAddr
}

func (srv adminServer) GrpcAddr() string {
	return srv.grpcAddr
}

func (adminServer) StaticFiles() http.FileSystem {
	return http.FS(gDocsFiles)
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

func (adminServer) Platform(ctx context.Context, pa *Void) (*Platform, error) {
	if gCurrentRtPlat != nil {
		return gCurrentRtPlat.Platform, nil
	}
	return nil, status.New(codes.FailedPrecondition, "platform not yet initialized").Err()
}

func (srv adminServer) ConfigRead(ctx context.Context, pa *Void) (*ConfigReadResponse, error) {
	return srv.confMgr.BuildConfigResponse(), nil
}

func (adminServer) Producers(ctx context.Context, pa *Void) (*TrackedProducers, error) {
	return gProducerTracker.toTrackedProducers(), nil
}

func (adminServer) DecodeInstance(ctx context.Context, args *DecodeInstanceArgs) (*DecodeResponse, error) {
	dec, err := decodeInstance64(ctx, args.Concern, args.Payload64)
	if err != nil {
		return nil, err
	}
	return &DecodeResponse{
		Json: dec,
	}, nil
}

func (adminServer) DecodeArgPayload(ctx context.Context, args *DecodePayloadArgs) (*DecodeResponse, error) {
	dec, err := decodeArgPayload64(ctx, args.Concern, args.System, args.Command, args.Payload64)
	if err != nil {
		return nil, err
	}
	return &DecodeResponse{
		Json: dec,
	}, nil
}

func (adminServer) DecodeResultPayload(ctx context.Context, args *DecodePayloadArgs) (*DecodeResponse, error) {
	dec, err := decodeResultPayload64(ctx, args.Concern, args.System, args.Command, args.Payload64)
	if err != nil {
		return nil, err
	}
	return &DecodeResponse{
		Json: dec,
	}, nil
}

func adminServe(ctx context.Context, httpAddr string, grpcAddr string, wg *sync.WaitGroup) {
	srv := adminServer{
		httpAddr: httpAddr,
		grpcAddr: grpcAddr,
		confMgr:  NewConfigMgr(ctx, gSettings.AdminBrokers, PlatformName(), wg),
	}
	ServeGrpcGateway(ctx, srv)
}

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
		Str("Cluster", ci.cluster.Name).
		Str("Topic", name).
		Int("NumPartitions", numPartitions).
		Int("ReplicationFactor", replicationFactor).
		Msg("Topic created")

	return nil
}

func newClusterInfo(cluster *Platform_Cluster) (*clusterInfo, error) {
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

func createMissingTopic(topicName string, topic *Platform_Concern_Topic, clusterInfos map[string]*clusterInfo) {
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

func managePlatform(ctx context.Context, platformName string, wg *sync.WaitGroup) {
	adminTopic := AdminTopic(platformName)
	adminProdCh := getProducerCh(ctx, gSettings.AdminBrokers, wg)

	adminCh := make(chan *AdminMessage)
	wg.Add(1)
	go consumeAdminTopic(
		ctx,
		adminCh,
		gSettings.AdminBrokers,
		platformName,
		Directive_PLATFORM|Directive_PRODUCER_STATUS,
		Directive_PLATFORM,
		kAtLastMatch,
		wg,
	)

	republishTicker := time.NewTicker(gPlatformRepublishInterval)

	cullInterval := gAdminPingInterval * 10
	cullTicker := time.NewTicker(cullInterval)

	for {
		select {
		case <-ctx.Done():
			log.Warn().
				Msg("managePlatform exiting, ctx.Done()")
			return
		case <-republishTicker.C:
			if gCurrentRtPlat != nil {
				log.Info().Msg("Republishing platform")
				msg, err := kafkaMessage(&adminTopic, 0, gCurrentRtPlat.Platform, Directive_PLATFORM, "")
				if err != nil {
					log.Error().
						Err(err).
						Msg("Failed to kafkaMessage")
					continue
				}
				adminProdCh <- msg
			}
		case <-cullTicker.C:
			// cull stale producers
			gProducerTracker.cull(cullInterval)
		case adminMsg := <-adminCh:
			if (adminMsg.Directive & Directive_PLATFORM) == Directive_PLATFORM {
				gCurrentRtPlat = adminMsg.NewRtPlat

				jsonBytes, _ := protojson.Marshal(proto.Message(gCurrentRtPlat.Platform))
				log.Info().
					Str("PlatformJson", string(jsonBytes)).
					Msg("Platform Replaced")

				platDiff := gCurrentRtPlat.diff(adminMsg.OldRtPlat)
				updateTopics(gCurrentRtPlat)
				updateRunner(ctx, adminProdCh, adminTopic, platDiff)
			} else if (adminMsg.Directive & Directive_PRODUCER_STATUS) == Directive_PRODUCER_STATUS {
				gProducerTracker.update(adminMsg.ProducerDirective, adminMsg.Timestamp, gAdminPingInterval*2)
			}
		}
	}
}

func updateRunner(ctx context.Context, adminProdCh ProducerCh, adminTopic string, platDiff *platformDiff) {
	ctx, span := Telem().StartFunc(ctx)
	defer span.End()
	traceParent := ExtractTraceParent(ctx)

	for _, p := range platDiff.progsToStop {
		msg, err := kafkaMessage(
			&adminTopic,
			0,
			&ConsumerDirective{Program: p},
			Directive_CONSUMER_STOP,
			traceParent,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failure in kafkaMessage during updateRunner")
			return
		}

		adminProdCh <- msg
	}

	for _, p := range platDiff.progsToStart {
		msg, err := kafkaMessage(
			&adminTopic,
			0,
			&ConsumerDirective{Program: p},
			Directive_CONSUMER_START,
			traceParent,
		)
		if err != nil {
			log.Error().Err(err).Msg("Failure in kafkaMessage during updateRunner")
			return
		}

		adminProdCh <- msg
	}
}

func substStr(s string, concernName string, consumerBrokers string, shortTopicName string, fullTopicName string, partition int32) string {
	s = strings.ReplaceAll(s, "@platform", gPlatformName)
	s = strings.ReplaceAll(s, "@otelcol_endpoint", gSettings.OtelcolEndpoint)
	s = strings.ReplaceAll(s, "@admin_brokers", gSettings.AdminBrokers)
	s = strings.ReplaceAll(s, "@consumer_brokers", consumerBrokers)
	s = strings.ReplaceAll(s, "@concern", concernName)
	s = strings.ReplaceAll(s, "@system", shortTopicName)
	s = strings.ReplaceAll(s, "@topic", fullTopicName)
	s = strings.ReplaceAll(s, "@partition", strconv.Itoa(int(partition)))
	return s
}

var gStdTags map[string]string = map[string]string{
	"service.name":   "rkcy.@platform.@concern.@system",
	"rkcy.concern":   "@concern",
	"rkcy.system":    "@system",
	"rkcy.topic":     "@topic",
	"rkcy.partition": "@partition",
}

func expandProgs(concern *Platform_Concern, topics *Platform_Concern_Topics, clusters map[string]*Platform_Cluster) []*Program {
	progs := make([]*Program, topics.Current.PartitionCount)
	for i := int32(0); i < topics.Current.PartitionCount; i++ {
		topicName := BuildFullTopicName(gPlatformName, concern.Name, concern.Type, topics.Name, topics.Current.Generation)
		cluster := clusters[topics.Current.Cluster]
		progs[i] = &Program{
			Name:   substStr(topics.ConsumerProgram.Name, concern.Name, cluster.Brokers, topics.Name, topicName, i),
			Args:   make([]string, len(topics.ConsumerProgram.Args)),
			Abbrev: substStr(topics.ConsumerProgram.Abbrev, concern.Name, cluster.Brokers, topics.Name, topicName, i),
			Tags:   make(map[string]string),
		}
		for j := 0; j < len(topics.ConsumerProgram.Args); j++ {
			progs[i].Args[j] = substStr(topics.ConsumerProgram.Args[j], concern.Name, cluster.Brokers, topics.Name, topicName, i)
		}

		for k, v := range gStdTags {
			progs[i].Tags[k] = substStr(v, concern.Name, cluster.Brokers, topics.Name, topicName, i)
		}
		if topics.ConsumerProgram.Tags != nil {
			for k, v := range topics.ConsumerProgram.Tags {
				progs[i].Tags[k] = substStr(v, concern.Name, cluster.Brokers, topics.Name, topicName, i)
			}
		}
	}
	return progs
}
