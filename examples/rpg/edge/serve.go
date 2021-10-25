// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package edge

import (
	"context"
	"embed"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/version"

	"github.com/lachlanorr/rocketcycle/examples/rpg/pb"
	"github.com/lachlanorr/rocketcycle/examples/rpg/txn"
)

//go:embed static/docs
var docsFiles embed.FS

var (
	aprod *rkcy.ApecsProducer
)

type server struct {
	UnimplementedRpgServiceServer

	httpAddr string
	grpcAddr string

	wg *sync.WaitGroup
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
	RegisterRpgServiceServer(srvReg, srv)
}

func (server) RegisterHandlerFromEndpoint(
	ctx context.Context,
	mux *runtime.ServeMux,
	endpoint string,
	opts []grpc.DialOption,
) (err error) {
	return RegisterRpgServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

func timeout() time.Duration {
	return time.Duration(settings.TimeoutSecs) * time.Second
}

func (srv server) ReadPlayer(ctx context.Context, req *RpgRequest) (*PlayerResponse, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := aprod.ExecuteTxnSync(
		ctx,
		txn.ReadPlayer(req.Id),
		timeout(),
		srv.wg,
	)
	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &PlayerResponse{Player: resProto.Instance.(*pb.Player), Related: resProto.Related.(*pb.PlayerRelatedConcerns)}, nil
}

func (srv server) CreatePlayer(ctx context.Context, plyr *pb.Player) (*pb.Player, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := aprod.ExecuteTxnSync(
		ctx,
		txn.CreatePlayer(plyr),
		timeout(),
		srv.wg,
	)
	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resProto.Instance.(*pb.Player), nil
}

func (srv server) UpdatePlayer(ctx context.Context, plyr *pb.Player) (*pb.Player, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := aprod.ExecuteTxnSync(
		ctx,
		txn.UpdatePlayer(plyr),
		timeout(),
		srv.wg,
	)
	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resProto.Instance.(*pb.Player), nil
}

func (srv server) DeletePlayer(ctx context.Context, req *RpgRequest) (*pb.Player, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := aprod.ExecuteTxnSync(
		ctx,
		txn.DeletePlayer(req.Id),
		timeout(),
		srv.wg,
	)
	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resProto.Instance.(*pb.Player), nil
}

func (srv server) ReadCharacter(ctx context.Context, req *RpgRequest) (*CharacterResponse, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := aprod.ExecuteTxnSync(
		ctx,
		txn.ReadCharacter(req.Id),
		timeout(),
		srv.wg,
	)
	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &CharacterResponse{Character: resProto.Instance.(*pb.Character), Related: resProto.Related.(*pb.CharacterRelatedConcerns)}, nil
}

func (srv server) CreateCharacter(ctx context.Context, plyr *pb.Character) (*pb.Character, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := aprod.ExecuteTxnSync(
		ctx,
		txn.CreateCharacter(plyr),
		timeout(),
		srv.wg,
	)
	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resProto.Instance.(*pb.Character), nil
}

func (srv server) UpdateCharacter(ctx context.Context, plyr *pb.Character) (*pb.Character, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := aprod.ExecuteTxnSync(
		ctx,
		txn.UpdateCharacter(plyr),
		timeout(),
		srv.wg,
	)
	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resProto.Instance.(*pb.Character), nil
}

func (srv server) DeleteCharacter(ctx context.Context, req *RpgRequest) (*pb.Character, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := aprod.ExecuteTxnSync(
		ctx,
		txn.DeleteCharacter(req.Id),
		timeout(),
		srv.wg,
	)
	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resProto.Instance.(*pb.Character), nil
}

func (srv server) FundCharacter(ctx context.Context, fr *pb.FundingRequest) (*pb.Character, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := aprod.ExecuteTxnSync(
		ctx,
		txn.Fund(fr),
		timeout(),
		srv.wg,
	)
	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resProto.Instance.(*pb.Character), nil
}

func (srv server) ConductTrade(ctx context.Context, tr *pb.TradeRequest) (*rkcy.Void, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	_, err := aprod.ExecuteTxnSync(
		ctx,
		txn.Trade(tr),
		timeout(),
		srv.wg,
	)
	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &rkcy.Void{}, nil
}

func serve(
	ctx context.Context,
	httpAddr string,
	grpcAddr string,
	platformName string,
	wg *sync.WaitGroup,
) {
	srv := server{
		httpAddr: httpAddr,
		grpcAddr: grpcAddr,
		wg:       wg,
	}
	rkcy.ServeGrpcGateway(ctx, srv)
}

func cobraServe(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("edge server started")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	defer func() {
		signal.Stop(interruptCh)
		cancel()
	}()

	var wg sync.WaitGroup
	aprod = rkcy.NewApecsProducer(
		ctx,
		settings.AdminBrokers,
		rkcy.PlatformName(),
		&rkcy.TopicTarget{
			Brokers:   settings.ConsumerBrokers,
			Topic:     settings.Topic,
			Partition: settings.Partition,
		},
		&wg,
	)

	if aprod == nil {
		log.Fatal().
			Msg("Failed to NewApecsProducer")
	}

	go serve(ctx, settings.HttpAddr, settings.GrpcAddr, rkcy.PlatformName(), &wg)

	select {
	case <-interruptCh:
		log.Info().
			Msg("edge server stopped")
		cancel()
		wg.Wait()
		return
	}
}
