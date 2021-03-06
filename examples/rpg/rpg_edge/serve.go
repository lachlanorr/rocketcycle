// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"embed"
	"net/http"
	"os"
	"os/signal"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rkcy/examples/rpg/lib/commands"
	"github.com/lachlanorr/rkcy/pkg/rkcy"

	rpg_pb "github.com/lachlanorr/rkcy/examples/rpg/pb"
	rkcy_pb "github.com/lachlanorr/rkcy/pkg/rkcy/pb"
	"github.com/lachlanorr/rkcy/version"
)

//go:embed static/docs
var docsFiles embed.FS

var (
	prods map[string]*rkcy.ApecsProducer
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
	if in.Username == "" {
		return nil, status.New(codes.InvalidArgument, "missing username in create call").Err()
	}

	if in.Id != "" {
		return nil, status.New(codes.InvalidArgument, "id provided in create call").Err()
	}
	in.Id = uuid.NewString()

	inSer, err := proto.Marshal(in)
	if err != nil {
		return nil, status.New(codes.Internal, "failed to marshal Player").Err()
	}

	prod, ok := prods["player"]
	if !ok {
		return nil, status.New(codes.Internal, "no producer for 'player'").Err()
	}

	id := uuid.NewString()

	txn := rkcy_pb.ApecsTxn{
		Id:        id,
		CanRevert: true,
		ForwardSteps: []*rkcy_pb.ApecsTxn_Step{
			{
				ConcernName: "player",
				Command:     commands.Create,
				Key:         id,
				Payload:     inSer,
			},
		},
	}
	txnSer, err := proto.Marshal(&txn)
	if err != nil {
		return nil, status.New(codes.Internal, "failed to marshal ApecsTxn").Err()
	}

	prod.Process([]byte(txn.ForwardSteps[0].Key), txnSer)

	return in, nil
}

func enrollProducer(ctx context.Context, platformName string, concernName string) {
	prods[concernName] = rkcy.NewApecsProducer(ctx, bootstrapServers, platformName, concernName)
	if prods[concernName] == nil {
		log.Fatal().
			Msgf("Failure creating producer for '%s'", concernName)
	}
}

func serve(ctx context.Context, httpAddr string, grpcAddr string, platformName string) {
	prods = make(map[string]*rkcy.ApecsProducer)

	enrollProducer(ctx, platformName, "player")

	srv := server{httpAddr: httpAddr, grpcAddr: grpcAddr}
	rkcy.ServeGrpcGateway(ctx, srv)
}

func cobraServe(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("rcedge started")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go serve(ctx, httpAddr, grpcAddr, rkcy.PlatformName())

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	select {
	case <-interruptCh:
		cancel()
		return
	}
}
