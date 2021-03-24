// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package edge

import (
	"context"
	"embed"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	//"github.com/lachlanorr/rocketcycle/examples/rpg/commands"
	"github.com/lachlanorr/rocketcycle/examples/rpg/consts"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	rkcy_pb "github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"

	rpg_pb "github.com/lachlanorr/rocketcycle/examples/rpg/pb"
	"github.com/lachlanorr/rocketcycle/version"
)

//go:embed static/docs
var docsFiles embed.FS

var (
	aprod *rkcy.ApecsProducer
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

	reqId, err := aprod.ExecuteTxn(
		settings.Topic,
		settings.Partition,
		false,
		inSer,
		[]rkcy.Step{
			{
				ConcernName: consts.Player,
				Command:     rkcy_pb.Command_VALIDATE,
				Key:         in.Id,
			},
			{
				ConcernName: consts.Player,
				Command:     rkcy_pb.Command_CREATE,
				Key:         in.Id,
			},
		},
	)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to ExecuteTxn")
		return nil, status.New(codes.Internal, "failed to ExecuteTxn").Err()
	}

	log.Info().
		Str("ReqId", reqId).
		Msg("CreatePlayer")

	rspCh := make(chan *rkcy_pb.ApecsTxn)
	timer := time.NewTimer(2 * time.Second)
	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("edge server request exiting, ctx.Done()")
			return nil, status.New(codes.Internal, "context closed").Err()
		case <-timer.C:
			log.Fatal().
				Msg("edge server request exiting, ctx.Done()")
			return nil, status.New(codes.DeadlineExceeded, "time out waiting on response").Err()
		case txn := <-rspCh:
			txnJson := protojson.Format(proto.Message(txn))
			fmt.Println(txnJson)
			return in, nil
		}
	}

}

func serve(ctx context.Context, httpAddr string, grpcAddr string, platformName string) {
	srv := server{httpAddr: httpAddr, grpcAddr: grpcAddr}
	rkcy.ServeGrpcGateway(ctx, srv)
}

func cobraServe(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("edge server started")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	aprod = rkcy.NewApecsProducer(ctx, settings.BootstrapServers, rkcy.PlatformName())
	if aprod == nil {
		log.Fatal().
			Msg("Failed to NewApecsProducer")
	}

	go serve(ctx, settings.HttpAddr, settings.GrpcAddr, rkcy.PlatformName())

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	select {
	case <-interruptCh:
		log.Info().
			Msg("edge server stopped")
		cancel()
		return
	}
}
