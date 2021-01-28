// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"embed"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	admin_pb "github.com/lachlanorr/rocketcycle/build/proto/admin"
	"github.com/lachlanorr/rocketcycle/internal/rkcy"
)

//go:embed __static/docs
var docsFiles embed.FS

type server struct {
	admin_pb.UnimplementedAdminServiceServer

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
	admin_pb.RegisterAdminServiceServer(srvReg, srv)
}

func (server) RegisterHandlerFromEndpoint(
	ctx context.Context,
	mux *runtime.ServeMux,
	endpoint string,
	opts []grpc.DialOption,
) (err error) {
	return admin_pb.RegisterAdminServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

func (server) Platform(ctx context.Context, in *admin_pb.PlatformArgs) (*admin_pb.Platform, error) {
	if oldRtPlat != nil {
		return oldRtPlat.Platform, nil
	}
	return nil, status.New(codes.FailedPrecondition, "platform not yet initialized").Err()
}

func serve(ctx context.Context, httpAddr string, grpcAddr string) {
	srv := server{httpAddr: httpAddr, grpcAddr: grpcAddr}
	rkcy.Serve(ctx, srv)
}
