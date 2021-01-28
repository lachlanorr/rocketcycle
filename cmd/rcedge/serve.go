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

	admin_pb "github.com/lachlanorr/rocketcycle/build/proto/admin"
	"github.com/lachlanorr/rocketcycle/build/proto/edge"
	edge_pb "github.com/lachlanorr/rocketcycle/build/proto/edge"
	process_pb "github.com/lachlanorr/rocketcycle/build/proto/process"
	storage_pb "github.com/lachlanorr/rocketcycle/build/proto/storage"
	"github.com/lachlanorr/rocketcycle/internal/platform"
	"github.com/lachlanorr/rocketcycle/internal/rckafka"
	"github.com/lachlanorr/rocketcycle/internal/serve_utils"
)

//go:embed __static/docs
var docsFiles embed.FS

var (
	apecsProdCh = make(chan process_pb.ApecsTxn)
)

type server struct {
	edge.UnimplementedMmoServiceServer

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

func (srv server) RegisterServer(srvReg grpc.ServiceRegistrar) {
	edge.RegisterMmoServiceServer(srvReg, srv)
}

func (server) RegisterHandlerFromEndpoint(
	ctx context.Context,
	mux *runtime.ServeMux,
	endpoint string,
	opts []grpc.DialOption,
) (err error) {
	return edge.RegisterMmoServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

func (server) GetPlayer(ctx context.Context, in *edge_pb.MmoRequest) (*storage_pb.Player, error) {
	log.Info().Msg("GetPlayer " + in.Id)
	player := storage_pb.Player{}
	return &player, nil
}

func (server) CreatePlayer(ctx context.Context, in *storage_pb.Player) (*storage_pb.Player, error) {
	if in.Id != "" {
		return nil, status.New(codes.InvalidArgument, "id provided in create call").Err()
	}
	log.Info().Msg("CreatePlayer " + in.Username)
	return in, nil
}

func manageProducers(ctx context.Context, bootstrapServers string, platformName string) {
	//producers := make(map[string]*kafka.Producer)

	platCh := make(chan admin_pb.Platform)
	go rckafka.ConsumePlatformConfig(ctx, platCh, bootstrapServers, platformName)

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("manageProducers exiting, ctx.Done()")
			return
		case plat := <-platCh:
			rtPlat, err := platform.NewRtPlatform(&plat)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to NewRtPlatform")
				continue
			}
			// apply producer changes for apecs apps
			for _, app := range plat.Apps {
				if app.Type == admin_pb.Platform_App_APECS {
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

		}
	}
}

func serve(ctx context.Context, httpAddr string, grpcAddr string) {
	srv := server{httpAddr: httpAddr, grpcAddr: grpcAddr}

	serve_utils.Serve(ctx, srv)
}
