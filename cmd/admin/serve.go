// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"embed"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/lachlanorr/rocketcycle/build/proto/admin"
	"github.com/lachlanorr/rocketcycle/internal/serve_utils"
)

//go:embed __static/docs
var docsFiles embed.FS

type server struct {
	pb.UnimplementedAdminServiceServer
}

func (s *server) Metadata(ctx context.Context, in *pb.MetadataArgs) (*pb.Metadata, error) {
	if oldRtmeta != nil {
		return oldRtmeta.meta, nil
	}
	return nil, status.New(codes.FailedPrecondition, "metadata not yet initialized").Err()
}

func prepareGrpcServer(ctx context.Context) {
	lis, err := net.Listen("tcp", ":11372")
	if err != nil {
		log.Error().
			Msg("Unable to create grpc listener on tcp port 11372")
		return
	}
	srv := server{}
	grpcServer := grpc.NewServer()
	pb.RegisterAdminServiceServer(grpcServer, &srv)

	if err := grpcServer.Serve(lis); err != nil {
		log.Error().
			Str("Error", err.Error()).
			Msg("failed to serve admin grpc")
	}
}

func serve(ctx context.Context) {
	// Start grpc server
	go prepareGrpcServer(ctx)

	// Register grpc gateway server endpoint
	apiMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := pb.RegisterAdminServiceHandlerFromEndpoint(ctx, apiMux, ":11372", opts)
	if err != nil {
		log.Error().
			Str("Error", err.Error()).
			Msg("pb.RegisterAdminServiceHandlerFromEndpoint failed")
		return
	}

	// Create our file system static + grpc mux wrapper
	fs := http.FS(docsFiles)
	mux, err := serve_utils.NewServeMux(fs, "^/docs", "/__static/docs", apiMux)
	if err != nil {
		log.Error().
			Str("Error", err.Error()).
			Msg("serve_utils.NewServerMux failed")
		return
	}

	// This listener will serve up swagger on /docs and pass all other
	// requests through to the grpc gw
	err = http.ListenAndServe(":11371", mux)
	if err != nil {
		log.Error().
			Str("Error", err.Error()).
			Msg("http.ListenAndServe failed")
		return
	}
}
