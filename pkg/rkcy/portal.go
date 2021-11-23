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

	"github.com/lachlanorr/rocketcycle/version"
)

//go:embed static/portal/docs
var gDocsFiles embed.FS

func (plat *Platform) cobraPortalServe(cmd *cobra.Command, args []string) {
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
	go portalServe(ctx, plat, &wg)

	go plat.portalPlatform(ctx, &wg)

	select {
	case <-interruptCh:
		log.Warn().
			Msg("portal server stopped")
		cancel()
		wg.Wait()
		return
	}
}

func (plat *Platform) cobraPortalReadPlatform(cmd *cobra.Command, args []string) {
	path := "/v1/platform/read?pretty"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(plat.settings.PortalAddr + path)
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

func (plat *Platform) cobraPortalReadConfig(cmd *cobra.Command, args []string) {
	path := "/v1/config/read"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(plat.settings.PortalAddr + path)
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

func (plat *Platform) cobraPortalReadProducers(cmd *cobra.Command, args []string) {
	path := "/v1/producers/read?pretty"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(plat.settings.PortalAddr + path)
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

func (plat *Platform) cobraPortalCancelTxn(cmd *cobra.Command, args []string) {
	conn, err := grpc.Dial(plat.settings.PortalAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatal().
			Err(err).
			Str("PortalAddr", plat.settings.PortalAddr).
			Msg("Failed to grpc.Dial")
	}
	defer conn.Close()
	client := NewPortalServiceClient(conn)

	cancelTxnReq := &CancelTxnRequest{
		TxnId: args[0],
	}

	_, err = client.CancelTxn(context.Background(), cancelTxnReq)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("CancelTxn error")
	}
}

func (plat *Platform) cobraPortalDecodeInstance(cmd *cobra.Command, args []string) {
	path := "/v1/instance/decode"
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
	resp, err := http.Post(plat.settings.PortalAddr+path, "application/json", contentRdr)
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

	fmt.Printf("%s\n", decodeRsp.Instance)
	if decodeRsp.Related != "" {
		fmt.Printf("\nRelated:\n%s\n", decodeRsp.Related)
	}
}

type portalServer struct {
	UnimplementedPortalServiceServer

	plat *Platform
}

func (srv portalServer) HttpAddr() string {
	return srv.plat.settings.HttpAddr
}

func (srv portalServer) GrpcAddr() string {
	return srv.plat.settings.GrpcAddr
}

func (portalServer) StaticFiles() http.FileSystem {
	return http.FS(gDocsFiles)
}

func (portalServer) StaticFilesPathPrefix() string {
	return "/static/portal/docs"
}

func (srv portalServer) RegisterServer(srvReg grpc.ServiceRegistrar) {
	RegisterPortalServiceServer(srvReg, srv)
}

func (portalServer) RegisterHandlerFromEndpoint(
	ctx context.Context,
	mux *runtime.ServeMux,
	endpoint string,
	opts []grpc.DialOption,
) (err error) {
	return RegisterPortalServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

func (srv portalServer) PlatformDef(ctx context.Context, pa *Void) (*PlatformDef, error) {
	if srv.plat.currentRtPlatDef != nil {
		return srv.plat.currentRtPlatDef.PlatformDef, nil
	}
	return nil, status.Error(codes.FailedPrecondition, "platform not yet initialized")
}

func (srv portalServer) ConfigRead(ctx context.Context, pa *Void) (*ConfigReadResponse, error) {
	return srv.plat.ConfigMgr().BuildConfigResponse(), nil
}

func (srv portalServer) DecodeInstance(ctx context.Context, args *DecodeInstanceArgs) (*DecodeResponse, error) {
	jsonBytes, err := srv.plat.concernHandlers.decodeInstance64Json(ctx, args.Concern, args.Payload64)
	if err != nil {
		return nil, err
	}
	return &DecodeResponse{
		Type:     args.Concern,
		Instance: string(jsonBytes),
	}, nil
}

func resultProto2DecodeResponse(resProto *ResultProto) (*DecodeResponse, error) {
	instJson, err := protojson.Marshal(resProto.Instance)
	if err != nil {
		return nil, err
	}

	decResp := &DecodeResponse{
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

func (srv portalServer) DecodeArgPayload(ctx context.Context, args *DecodePayloadArgs) (*DecodeResponse, error) {
	resProto, _, err := srv.plat.concernHandlers.decodeArgPayload64(ctx, args.Concern, args.System, args.Command, args.Payload64)
	if err != nil {
		return nil, err
	}
	return resultProto2DecodeResponse(resProto)
}

func (srv portalServer) DecodeResultPayload(ctx context.Context, args *DecodePayloadArgs) (*DecodeResponse, error) {
	resProto, _, err := srv.plat.concernHandlers.decodeResultPayload64(ctx, args.Concern, args.System, args.Command, args.Payload64)
	if err != nil {
		return nil, err
	}
	return resultProto2DecodeResponse(resProto)
}

func (srv portalServer) CancelTxn(ctx context.Context, cancelTxn *CancelTxnRequest) (*Void, error) {
	ctx, traceId, span := srv.plat.telem.StartRequest(ctx)
	defer span.End()

	log.Warn().Msgf("CancelTxn %s", cancelTxn.TxnId)

	cncAdminDir := &ConcernAdminDirective{
		TxnId: cancelTxn.TxnId,
	}

	wg := &sync.WaitGroup{}

	platCh := make(chan *PlatformMessage)
	consumePlatformTopic(
		ctx,
		platCh,
		srv.plat.settings.AdminBrokers,
		srv.plat.name,
		srv.plat.environment,
		nil,
		wg,
	)

	platMsg := <-platCh
	rtPlat := platMsg.NewRtPlatDef

	for _, rtCnc := range rtPlat.Concerns {
		if rtCnc.Concern.Type == Concern_APECS {
			adminRtTopics, ok := rtCnc.Topics[string(ADMIN)]
			if !ok {
				return nil, fmt.Errorf("No admin topic for concern: %s", rtCnc.Concern.Name)
			}
			cluster, ok := rtPlat.Clusters[adminRtTopics.Topics.Current.Cluster]
			if !ok {
				return nil, fmt.Errorf("No brokers (%s) for concern topic: %s.%s", adminRtTopics.Topics.Current.Cluster, rtCnc.Concern.Name, ADMIN)
			}
			log.Info().Msgf("%s - %s", cluster.Brokers, adminRtTopics.CurrentTopic)

			prodCh := srv.plat.rawProducer.getProducerCh(ctx, cluster.Brokers, wg)
			msg, err := newKafkaMessage(
				&adminRtTopics.CurrentTopic,
				0,
				cncAdminDir,
				Directive_CONCERN_ADMIN_CANCEL_TXN,
				traceId,
			)
			if err != nil {
				return nil, err
			}
			prodCh <- msg
		}
	}

	return &Void{}, nil
}

func portalServe(ctx context.Context, plat *Platform, wg *sync.WaitGroup) {
	srv := portalServer{
		plat: plat,
	}
	plat.InitConfigMgr(ctx, wg)
	ServeGrpcGateway(ctx, srv)
}

func (plat *Platform) portalPlatform(
	ctx context.Context,
	wg *sync.WaitGroup,
) {
	platCh := make(chan *PlatformMessage)
	consumePlatformTopic(
		ctx,
		platCh,
		plat.settings.AdminBrokers,
		plat.name,
		plat.environment,
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
			if (platMsg.Directive & Directive_PLATFORM) != Directive_PLATFORM {
				log.Error().Msgf("Invalid directive for PlatformTopic: %s", platMsg.Directive.String())
				continue
			}

			plat.currentRtPlatDef = platMsg.NewRtPlatDef
		}
	}
}
