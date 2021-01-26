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

	"github.com/lachlanorr/rocketcycle/build/proto/edge"
	edge_pb "github.com/lachlanorr/rocketcycle/build/proto/edge"
	storage_pb "github.com/lachlanorr/rocketcycle/build/proto/storage"
	"github.com/lachlanorr/rocketcycle/internal/serve_utils"
)

//go:embed __static/docs
var docsFiles embed.FS

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
	log.Info().Msg("GetPlayer " + in.Uuid)
	player := storage_pb.Player{}
	return &player, nil
}

func (server) CreatePlayer(ctx context.Context, in *storage_pb.Player) (*storage_pb.Player, error) {
	log.Info().Msg("CreatePlayer " + in.Username)
	return in, nil
}

func serve(ctx context.Context, httpAddr string, grpcAddr string) {
	srv := server{httpAddr: httpAddr, grpcAddr: grpcAddr}
	serve_utils.Serve(ctx, srv)
}
