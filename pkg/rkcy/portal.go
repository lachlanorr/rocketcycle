// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/version"
)

//go:embed static/portal/docs
var gDocsFiles embed.FS

func (kplat *KafkaPlatform) cobraPortalServe(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("portal server started")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	defer func() {
		signal.Stop(interruptCh)
		cancel()
	}()

	var wg sync.WaitGroup
	go portalServe(ctx, kplat, &wg)

	go kplat.portalPlatform(ctx, &wg)

	select {
	case <-interruptCh:
		log.Warn().
			Msg("portal server stopped")
		cancel()
		wg.Wait()
		return
	}
}

func (kplat *KafkaPlatform) cobraPortalReadPlatform(cmd *cobra.Command, args []string) {
	path := "/v1/platform/read?pretty"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(kplat.settings.PortalAddr + path)
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

func (kplat *KafkaPlatform) cobraPortalReadConfig(cmd *cobra.Command, args []string) {
	path := "/v1/config/read"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(kplat.settings.PortalAddr + path)
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

	confRsp := &rkcypb.ConfigReadResponse{}
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

func (kplat *KafkaPlatform) cobraPortalReadProducers(cmd *cobra.Command, args []string) {
	path := "/v1/producers/read?pretty"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(kplat.settings.PortalAddr + path)
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

func (kplat *KafkaPlatform) cobraPortalCancelTxn(cmd *cobra.Command, args []string) {
	conn, err := grpc.Dial(kplat.settings.PortalAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatal().
			Err(err).
			Str("PortalAddr", kplat.settings.PortalAddr).
			Msg("Failed to grpc.Dial")
	}
	defer conn.Close()
	client := rkcypb.NewPortalServiceClient(conn)

	cancelTxnReq := &rkcypb.CancelTxnRequest{
		TxnId: args[0],
	}

	_, err = client.CancelTxn(context.Background(), cancelTxnReq)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("CancelTxn error")
	}
}

func (kplat *KafkaPlatform) cobraPortalDecodeInstance(cmd *cobra.Command, args []string) {
	path := "/v1/instance/decode"
	slog := log.With().
		Str("Path", path).
		Logger()

	var err error

	rpcArgs := rkcypb.DecodeInstanceArgs{
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
	resp, err := http.Post(kplat.settings.PortalAddr+path, "application/json", contentRdr)
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

	decodeRsp := rkcypb.DecodeResponse{}
	err = protojson.Unmarshal(body, &decodeRsp)
	if err != nil {
		slog.Fatal().
			Err(err).
			Msg("Failed to Unmarshal DecodeResponse")
	}

	fmt.Printf("%s\n", decodeRsp.Instance)
	if decodeRsp.Related != "" {
		fmt.Printf("\nRelated:\n%s\n", decodeRsp.Related)
	}
}

type portalServer struct {
	rkcypb.UnimplementedPortalServiceServer

	kplat *KafkaPlatform
}

func (srv portalServer) HttpAddr() string {
	return srv.kplat.settings.HttpAddr
}

func (srv portalServer) GrpcAddr() string {
	return srv.kplat.settings.GrpcAddr
}

func (portalServer) StaticFiles() http.FileSystem {
	return http.FS(gDocsFiles)
}

func (portalServer) StaticFilesPathPrefix() string {
	return "/static/portal/docs"
}

func (srv portalServer) RegisterServer(srvReg grpc.ServiceRegistrar) {
	rkcypb.RegisterPortalServiceServer(srvReg, srv)
}

func (portalServer) RegisterHandlerFromEndpoint(
	ctx context.Context,
	mux *runtime.ServeMux,
	endpoint string,
	opts []grpc.DialOption,
) (err error) {
	return rkcypb.RegisterPortalServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

func (srv portalServer) PlatformDef(ctx context.Context, pa *rkcypb.Void) (*rkcypb.PlatformDef, error) {
	if srv.kplat.currentRtPlatDef != nil {
		return srv.kplat.currentRtPlatDef.PlatformDef, nil
	}
	return nil, status.Error(codes.FailedPrecondition, "platform not yet initialized")
}

func (srv portalServer) ConfigRead(ctx context.Context, pa *rkcypb.Void) (*rkcypb.ConfigReadResponse, error) {
	return srv.kplat.ConfigMgr().BuildConfigResponse(), nil
}

func (srv portalServer) DecodeInstance(ctx context.Context, args *rkcypb.DecodeInstanceArgs) (*rkcypb.DecodeResponse, error) {
	jsonBytes, err := srv.kplat.concernHandlers.decodeInstance64Json(ctx, args.Concern, args.Payload64)
	if err != nil {
		return nil, err
	}
	return &rkcypb.DecodeResponse{
		Type:     args.Concern,
		Instance: string(jsonBytes),
	}, nil
}

func resultProto2DecodeResponse(resProto *ResultProto) (*rkcypb.DecodeResponse, error) {
	instJson, err := protojson.Marshal(resProto.Instance)
	if err != nil {
		return nil, err
	}

	decResp := &rkcypb.DecodeResponse{
		Type:     resProto.Type,
		Instance: string(instJson),
	}

	if resProto.Related != nil {
		relJson, err := protojson.Marshal(resProto.Related)
		if err != nil {
			return nil, err
		}
		decResp.Related = string(relJson)
	}

	return decResp, nil
}

func (srv portalServer) DecodeArgPayload(ctx context.Context, args *rkcypb.DecodePayloadArgs) (*rkcypb.DecodeResponse, error) {
	resProto, _, err := srv.kplat.concernHandlers.decodeArgPayload64(ctx, args.Concern, args.System, args.Command, args.Payload64)
	if err != nil {
		return nil, err
	}
	return resultProto2DecodeResponse(resProto)
}

func (srv portalServer) DecodeResultPayload(ctx context.Context, args *rkcypb.DecodePayloadArgs) (*rkcypb.DecodeResponse, error) {
	resProto, _, err := srv.kplat.concernHandlers.decodeResultPayload64(ctx, args.Concern, args.System, args.Command, args.Payload64)
	if err != nil {
		return nil, err
	}
	return resultProto2DecodeResponse(resProto)
}

func (srv portalServer) CancelTxn(ctx context.Context, cancelTxn *rkcypb.CancelTxnRequest) (*rkcypb.Void, error) {
	ctx, traceId, span := srv.kplat.telem.StartRequest(ctx)
	defer span.End()

	log.Warn().Msgf("CancelTxn %s", cancelTxn.TxnId)

	cncAdminDir := &rkcypb.ConcernAdminDirective{
		TxnId: cancelTxn.TxnId,
	}

	wg := &sync.WaitGroup{}

	platCh := make(chan *PlatformMessage)
	consumePlatformTopic(
		ctx,
		platCh,
		srv.kplat.settings.AdminBrokers,
		srv.kplat.name,
		srv.kplat.environment,
		nil,
		wg,
	)

	platMsg := <-platCh
	rtPlat := platMsg.NewRtPlatDef

	for _, rtCnc := range rtPlat.Concerns {
		if rtCnc.Concern.Type == rkcypb.Concern_APECS {
			adminRtTopics, ok := rtCnc.Topics[string(ADMIN)]
			if !ok {
				return nil, fmt.Errorf("No admin topic for concern: %s", rtCnc.Concern.Name)
			}
			cluster, ok := rtPlat.Clusters[adminRtTopics.Topics.Current.Cluster]
			if !ok {
				return nil, fmt.Errorf("No brokers (%s) for concern topic: %s.%s", adminRtTopics.Topics.Current.Cluster, rtCnc.Concern.Name, ADMIN)
			}
			log.Info().Msgf("%s - %s", cluster.Brokers, adminRtTopics.CurrentTopic)

			prodCh := srv.kplat.rawProducer.getProducerCh(ctx, cluster.Brokers, wg)
			msg, err := newKafkaMessage(
				&adminRtTopics.CurrentTopic,
				0,
				cncAdminDir,
				rkcypb.Directive_CONCERN_ADMIN_CANCEL_TXN,
				traceId,
			)
			if err != nil {
				return nil, err
			}
			prodCh <- msg
		}
	}

	return &rkcypb.Void{}, nil
}

func portalServe(ctx context.Context, kplat *KafkaPlatform, wg *sync.WaitGroup) {
	srv := portalServer{
		kplat: kplat,
	}
	kplat.InitConfigMgr(ctx, wg)
	ServeGrpcGateway(ctx, srv)
}

func (kplat *KafkaPlatform) portalPlatform(
	ctx context.Context,
	wg *sync.WaitGroup,
) {
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

	for {
		select {
		case <-ctx.Done():
			log.Warn().
				Msg("managePlatform exiting, ctx.Done()")
			return
		case platMsg := <-platCh:
			if (platMsg.Directive & rkcypb.Directive_PLATFORM) != rkcypb.Directive_PLATFORM {
				log.Error().Msgf("Invalid directive for PlatformTopic: %s", platMsg.Directive.String())
				continue
			}

			kplat.currentRtPlatDef = platMsg.NewRtPlatDef
		}
	}
}
