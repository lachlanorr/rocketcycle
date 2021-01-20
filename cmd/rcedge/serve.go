// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"embed"
	"flag"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"

	"github.com/lachlanorr/rocketcycle/build/proto/edge"
	"github.com/lachlanorr/rocketcycle/build/proto/storage"
	"github.com/lachlanorr/rocketcycle/internal/serve_utils"
)

//go:embed __static/docs
var docsFiles embed.FS

var httpAddr = flag.String("http_addr", ":11372", "Address for http listener")
var grpcAddr = flag.String("grpc_addr", ":11382", "Address for grpc listener")

type server struct {
	edge.UnimplementedMmoServiceServer
}

func (server) HttpAddr() string {
	return *httpAddr
}

func (server) GrpcAddr() string {
	return *grpcAddr
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

func (server) CreatePlayer(ctx context.Context, in *storage.Player) (*storage.Player, error) {
	player := storage.Player{}
	return &player, nil
}

func serve(ctx context.Context) {
	srv := server{}
	serve_utils.Serve(ctx, srv)
}
