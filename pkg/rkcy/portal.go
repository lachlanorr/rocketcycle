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

func cobraPortalServe(cmd *cobra.Command, args []string) {
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
	go portalServe(ctx, gSettings.HttpAddr, gSettings.GrpcAddr, &wg)

	select {
	case <-interruptCh:
		log.Warn().
			Msg("portal server stopped")
		cancel()
		wg.Wait()
		return
	}
}

func cobraPortalReadPlatform(cmd *cobra.Command, args []string) {
	path := "/v1/platform/read?pretty"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(gSettings.PortalAddr + path)
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

func cobraPortalReadConfig(cmd *cobra.Command, args []string) {
	path := "/v1/config/read"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(gSettings.PortalAddr + path)
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

func cobraPortalReadProducers(cmd *cobra.Command, args []string) {
	path := "/v1/producers/read?pretty"

	slog := log.With().
		Str("Path", path).
		Logger()

	resp, err := http.Get(gSettings.PortalAddr + path)
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

func cobraPortalDecodeInstance(cmd *cobra.Command, args []string) {
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
	resp, err := http.Post(gSettings.PortalAddr+path, "application/json", contentRdr)
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

	httpAddr string
	grpcAddr string

	confMgr *ConfigMgr
}

func (srv portalServer) HttpAddr() string {
	return srv.httpAddr
}

func (srv portalServer) GrpcAddr() string {
	return srv.grpcAddr
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

func (portalServer) Platform(ctx context.Context, pa *Void) (*Platform, error) {
	if gCurrentRtPlat != nil {
		return gCurrentRtPlat.Platform, nil
	}
	return nil, status.Error(codes.FailedPrecondition, "platform not yet initialized")
}

func (srv portalServer) ConfigRead(ctx context.Context, pa *Void) (*ConfigReadResponse, error) {
	return srv.confMgr.BuildConfigResponse(), nil
}

func (portalServer) Producers(ctx context.Context, pa *Void) (*TrackedProducers, error) {
	return gProducerTracker.toTrackedProducers(), nil
}

func (portalServer) DecodeInstance(ctx context.Context, args *DecodeInstanceArgs) (*DecodeResponse, error) {
	jsonBytes, err := decodeInstance64Json(ctx, args.Concern, args.Payload64)
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

func (portalServer) DecodeArgPayload(ctx context.Context, args *DecodePayloadArgs) (*DecodeResponse, error) {
	resProto, _, err := decodeArgPayload64(ctx, args.Concern, args.System, args.Command, args.Payload64)
	if err != nil {
		return nil, err
	}
	return resultProto2DecodeResponse(resProto)
}

func (portalServer) DecodeResultPayload(ctx context.Context, args *DecodePayloadArgs) (*DecodeResponse, error) {
	resProto, _, err := decodeResultPayload64(ctx, args.Concern, args.System, args.Command, args.Payload64)
	if err != nil {
		return nil, err
	}
	return resultProto2DecodeResponse(resProto)
}

func portalServe(ctx context.Context, httpAddr string, grpcAddr string, wg *sync.WaitGroup) {
	srv := portalServer{
		httpAddr: httpAddr,
		grpcAddr: grpcAddr,
		confMgr:  NewConfigMgr(ctx, gSettings.AdminBrokers, PlatformName(), Environment(), wg),
	}
	ServeGrpcGateway(ctx, srv)
}
