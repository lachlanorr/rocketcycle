// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package portal

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/lachlanorr/rocketcycle/pkg/mgmt"
	"github.com/lachlanorr/rocketcycle/pkg/platform"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/pkg/telem"
	"github.com/lachlanorr/rocketcycle/version"
)

//go:embed static/docs
var gDocsFiles embed.FS

type portalServer struct {
	rkcypb.UnimplementedPortalServiceServer

	plat     *platform.Platform
	httpAddr string
	grpcAddr string
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
	if srv.plat.RtPlatformDef != nil {
		return srv.plat.RtPlatformDef.PlatformDef, nil
	}
	return nil, status.Error(codes.FailedPrecondition, "platform not yet initialized")
}

func (srv portalServer) ConfigRead(ctx context.Context, pa *rkcypb.Void) (*rkcypb.ConfigReadResponse, error) {
	return srv.plat.ConfigRdr().BuildConfigResponse(), nil
}

func (srv portalServer) DecodeInstance(ctx context.Context, args *rkcypb.DecodeInstanceArgs) (*rkcypb.DecodeResponse, error) {
	jsonBytes, err := srv.plat.ConcernHandlers.DecodeInstance64Json(ctx, args.Concern, args.Payload64)
	if err != nil {
		return nil, err
	}
	return &rkcypb.DecodeResponse{
		Type:     args.Concern,
		Instance: string(jsonBytes),
	}, nil
}

func resultProto2DecodeResponse(resProto *rkcy.ResultProto) (*rkcypb.DecodeResponse, error) {
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
	resProto, _, err := srv.plat.ConcernHandlers.DecodeArgPayload64(ctx, args.Concern, args.System, args.Command, args.Payload64)
	if err != nil {
		return nil, err
	}
	return resultProto2DecodeResponse(resProto)
}

func (srv portalServer) DecodeResultPayload(ctx context.Context, args *rkcypb.DecodePayloadArgs) (*rkcypb.DecodeResponse, error) {
	resProto, _, err := srv.plat.ConcernHandlers.DecodeResultPayload64(ctx, args.Concern, args.System, args.Command, args.Payload64)
	if err != nil {
		return nil, err
	}
	return resultProto2DecodeResponse(resProto)
}

func (srv portalServer) CancelTxn(ctx context.Context, cancelTxn *rkcypb.CancelTxnRequest) (*rkcypb.Void, error) {
	ctx, traceId, span := telem.StartRequest(ctx)
	defer span.End()

	log.Warn().Msgf("CancelTxn %s", cancelTxn.TxnId)

	cncAdminDir := &rkcypb.ConcernAdminDirective{
		TxnId: cancelTxn.TxnId,
	}

	wg := &sync.WaitGroup{}

	platCh := make(chan *mgmt.PlatformMessage)
	mgmt.ConsumePlatformTopic(
		ctx,
		wg,
		srv.plat.StreamProvider(),
		srv.plat.Args.Platform,
		srv.plat.Args.Environment,
		srv.plat.Args.AdminBrokers,
		platCh,
		nil,
	)

	platMsg := <-platCh
	rtPlat := platMsg.NewRtPlatDef

	for _, rtCnc := range rtPlat.Concerns {
		if rtCnc.Concern.Type == rkcypb.Concern_APECS {
			adminRtTopics, ok := rtCnc.Topics[string(rkcy.ADMIN)]
			if !ok {
				return nil, fmt.Errorf("No admin topic for concern: %s", rtCnc.Concern.Name)
			}
			cluster, ok := rtPlat.Clusters[adminRtTopics.Topics.Current.Cluster]
			if !ok {
				return nil, fmt.Errorf("No brokers (%s) for concern topic: %s.%s", adminRtTopics.Topics.Current.Cluster, rtCnc.Concern.Name, rkcy.ADMIN)
			}
			log.Info().Msgf("%s - %s", cluster.Brokers, adminRtTopics.CurrentTopic)

			prodCh := srv.plat.GetProducerCh(ctx, wg, cluster.Brokers)
			msg, err := rkcy.NewKafkaMessage(
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

func serveGrpc(
	ctx context.Context,
	plat *platform.Platform,
	httpAddr string,
	grpcAddr string,
) {
	srv := portalServer{
		plat:     plat,
		httpAddr: httpAddr,
		grpcAddr: grpcAddr,
	}
	plat.InitConfigMgr(ctx)
	ServeGrpcGateway(ctx, srv)
}

func Serve(
	ctx context.Context,
	plat *platform.Platform,
	httpAddr string,
	grpcAddr string,
) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("portal server started")

	go serveGrpc(ctx, plat, httpAddr, grpcAddr)

	go managePlatform(ctx, plat)

	plat.WaitGroup().Add(1)
	select {
	case <-ctx.Done():
		log.Info().
			Msg("portal stopped")
		plat.WaitGroup().Done()
		return
	}
}

func managePlatform(
	ctx context.Context,
	plat *platform.Platform,
) {
	platCh := make(chan *mgmt.PlatformMessage)
	mgmt.ConsumePlatformTopic(
		ctx,
		plat.WaitGroup(),
		plat.StreamProvider(),
		plat.Args.Platform,
		plat.Args.Environment,
		plat.Args.AdminBrokers,
		platCh,
		nil,
	)

	plat.WaitGroup().Add(1)
	for {
		select {
		case <-ctx.Done():
			log.Warn().
				Msg("managePlatform exiting, ctx.Done()")
			plat.WaitGroup().Done()
			return
		case platMsg := <-platCh:
			if (platMsg.Directive & rkcypb.Directive_PLATFORM) != rkcypb.Directive_PLATFORM {
				log.Error().Msgf("Invalid directive for PlatformTopic: %s", platMsg.Directive.String())
				continue
			}

			plat.RtPlatformDef = platMsg.NewRtPlatDef
		}
	}
}
