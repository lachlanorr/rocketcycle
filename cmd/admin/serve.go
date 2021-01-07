// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"embed"
	"flag"
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

var httpAddr = flag.String("http_addr", ":11371", "Address for http listener")
var grpcAddr = flag.String("grpc_addr", ":11372", "Address for grpc listener")

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
	lis, err := net.Listen("tcp", *grpcAddr)
	if err != nil {
		log.Error().
			Msg("Unable to create grpc listener on tcp port 11372")
		return
	}
	srv := server{}
	grpcServer := grpc.NewServer()
	pb.RegisterAdminServiceServer(grpcServer, &srv)

	log.Info().
		Str("Address", *grpcAddr).
		Msg("gRPC server started")

	if err := grpcServer.Serve(lis); err != nil {
		log.Error().
			Str("Error", err.Error()).
			Msg("failed to serve admin grpc")
		return
	}
}

func serve(ctx context.Context) {
	// Start grpc server
	go prepareGrpcServer(ctx)

	// Register grpc gateway server endpoint
	apiMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := pb.RegisterAdminServiceHandlerFromEndpoint(ctx, apiMux, *grpcAddr, opts)
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

	log.Info().
		Str("Address", *httpAddr).
		Msg("HTTP server started")

	// This listener will serve up swagger on /docs and pass all other
	// requests through to the grpc gw
	err = http.ListenAndServe(*httpAddr, mux)
	if err != nil {
		log.Error().
			Str("Error", err.Error()).
			Msg("http.ListenAndServe failed")
		return
	}
}
