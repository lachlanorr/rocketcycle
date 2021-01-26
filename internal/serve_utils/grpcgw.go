// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package serve_utils

import (
	"context"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

type grpcgw interface {
	HttpAddr() string
	GrpcAddr() string

	StaticFiles() http.FileSystem

	RegisterServer(s grpc.ServiceRegistrar)
	RegisterHandlerFromEndpoint(
		ctx context.Context,
		mux *runtime.ServeMux,
		endpoint string,
		opts []grpc.DialOption,
	) (err error)
}

func prepareGrpcServer(ctx context.Context, srv interface{}) {
	lis, err := net.Listen("tcp", srv.(grpcgw).GrpcAddr())
	if err != nil {
		log.Error().
			Msg("Unable to create grpc listener on tcp port 11372")
		return
	}
	grpcServer := grpc.NewServer()
	srv.(grpcgw).RegisterServer(grpcServer)

	log.Info().
		Str("Address", srv.(grpcgw).GrpcAddr()).
		Msg("gRPC server started")

	if err := grpcServer.Serve(lis); err != nil {
		log.Error().
			Err(err).
			Msg("failed to serve grpc")
		return
	}
}

func prettyHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// checking Values as map[string][]string also catches ?pretty and ?pretty=
		// r.URL.Query().Get("pretty") would not.
		if _, ok := r.URL.Query()["pretty"]; ok {
			r.Header.Set("Accept", "application/json+pretty")
		}
		h.ServeHTTP(w, r)
	})
}

func Serve(ctx context.Context, srv interface{}) {
	// Start grpc server
	go prepareGrpcServer(ctx, srv)

	// Register grpc gateway server endpoint
	apiMux := runtime.NewServeMux(
		runtime.WithMarshalerOption("application/json+pretty", &runtime.JSONPb{
			MarshalOptions: protojson.MarshalOptions{
				Indent:          "  ",
				Multiline:       true,
				EmitUnpopulated: true,
			},
			UnmarshalOptions: protojson.UnmarshalOptions{
				DiscardUnknown: false,
			},
		}),
	)

	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := srv.(grpcgw).RegisterHandlerFromEndpoint(ctx, apiMux, srv.(grpcgw).GrpcAddr(), opts)
	if err != nil {
		log.Error().
			Err(err).
			Msg("RegisterHandlerFromEndpoint failed")
		return
	}

	// Create our file system static + grpc mux wrapper
	mux, err := NewServeMux(srv.(grpcgw).StaticFiles(), "^/docs", "/__static/docs", apiMux)
	if err != nil {
		log.Error().
			Err(err).
			Msg("serve_utils.NewServerMux failed")
		return
	}

	log.Info().
		Str("Address", srv.(grpcgw).HttpAddr()).
		Msg("HTTP server started")

	// This listener will serve up swagger on /docs and pass all other
	// requests through to the grpc gw
	err = http.ListenAndServe(srv.(grpcgw).HttpAddr(), prettyHandler(mux))
	if err != nil {
		log.Error().
			Err(err).
			Msg("http.ListenAndServe failed")
		return
	}
}
