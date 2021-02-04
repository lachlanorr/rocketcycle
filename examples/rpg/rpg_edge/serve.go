// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"embed"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"

	rpg_pb "github.com/lachlanorr/rocketcycle/examples/rpg/pb"
	rkcy_pb "github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
)

//go:embed static/docs
var docsFiles embed.FS

var (
	apecsProdCh = make(chan rkcy_pb.ApecsTxn)
)

type server struct {
	rpg_pb.UnimplementedMmoServiceServer

	httpAddr string
	grpcAddr string
}

func (srv server) HttpAddr() string {
	return srv.httpAddr
}

func (srv server) GrpcAddr() string {
	return srv.grpcAddr
}

func (server) StaticFiles() http.FileSystem {
	return http.FS(docsFiles)
}

func (server) StaticFilesPathPrefix() string {
	return "/static/docs"
}

func (srv server) RegisterServer(srvReg grpc.ServiceRegistrar) {
	rpg_pb.RegisterMmoServiceServer(srvReg, srv)
}

func (server) RegisterHandlerFromEndpoint(
	ctx context.Context,
	mux *runtime.ServeMux,
	endpoint string,
	opts []grpc.DialOption,
) (err error) {
	return rpg_pb.RegisterMmoServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

func (server) GetPlayer(ctx context.Context, in *rpg_pb.MmoRequest) (*rpg_pb.Player, error) {
	log.Info().Msg("GetPlayer " + in.Id)
	player := rpg_pb.Player{}
	return &player, nil
}

func (server) CreatePlayer(ctx context.Context, in *rpg_pb.Player) (*rpg_pb.Player, error) {
	if in.Id != "" {
		return nil, status.New(codes.InvalidArgument, "id provided in create call").Err()
	}
	log.Info().Msg("CreatePlayer " + in.Username)
	return in, nil
}

func manageProducers(ctx context.Context, bootstrapServers string, platformName string) {
	//producers := make(map[string]*kafka.Producer)

	platCh := make(chan rkcy_pb.Platform)
	go rkcy.ConsumePlatformConfig(ctx, platCh, bootstrapServers, platformName)

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("manageProducers exiting, ctx.Done()")
			return

			/* LORRTODO: create rkcy Producer and use it here, we don't care about platform messages directly
			case plat := <-platCh:

				rtPlat, err := rkcy.NewRtPlatform(&plat)
				if err != nil {
					log.Error().
						Err(err).
						Msg("Failed to NewRtPlatform")
					continue
				}
				// apply producer changes for apecs apps
				for _, app := range plat.Apps {
					if app.Type == rkcy_pb.Platform_App_APECS {
						topics := rtPlat.FindTopic(app.Name, "process")
						if topics == nil {
							log.Error().
								Msgf("No process topic in APECS app %s", app.Name)
							continue
						}

					}
				}
				// check for changes in apecs production targets

				//		case txn := <-apecsProdCh:
			*/
		}
	}
}

func serve(ctx context.Context, httpAddr string, grpcAddr string) {
	srv := server{httpAddr: httpAddr, grpcAddr: grpcAddr}

	rkcy.ServeGrpcGateway(ctx, srv)
}
