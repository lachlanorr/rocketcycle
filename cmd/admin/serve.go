// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	_ "embed"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/lachlanorr/rocketcycle/build/proto/admin"
)

//go:embed __static/swagger.json
var swaggerJson string

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
	lis, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Error().
			Msg("Unable to create grpc listener on tcp port 9000")
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

func listenAndServe(ctx context.Context) error {
	go prepareGrpcServer(ctx)

	// Register gRPC server endpoint
	// Note: Make sure the gRPC server is running properly and accessible
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := pb.RegisterAdminServiceHandlerFromEndpoint(ctx, mux, ":9000", opts)
	if err != nil {
		return err
	}

	// Start HTTP server (and proxy calls to gRPC server endpoint)
	return http.ListenAndServe(":8000", mux)
}
