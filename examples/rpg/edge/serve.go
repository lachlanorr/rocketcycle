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
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/version"

	"github.com/lachlanorr/rocketcycle/examples/rpg/logic/txn"
	"github.com/lachlanorr/rocketcycle/examples/rpg/pb"
)

//go:embed static/docs
var docsFiles embed.FS

var (
	aprod rkcy.ApecsProducer
)

type server struct {
	UnimplementedRpgServiceServer

	plat  rkcy.Platform
	aprod rkcy.ApecsProducer
	wg    *sync.WaitGroup
}

func (server) HttpAddr() string {
	return settings.HttpAddr
}

func (server) GrpcAddr() string {
	return settings.GrpcAddr
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
	ctx, span := srv.plat.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := rkcy.ExecuteTxnSync(
		ctx,
		srv.plat,
		srv.aprod,
		txn.ReadPlayer(req.Id),
		timeout(),
		srv.wg,
	)
	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &PlayerResponse{Player: resProto.Instance.(*pb.Player), Related: resProto.Related.(*pb.PlayerRelated)}, nil
}

func (srv server) CreatePlayer(ctx context.Context, plyr *pb.Player) (*pb.Player, error) {
	ctx, span := srv.plat.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := rkcy.ExecuteTxnSync(
		ctx,
		srv.plat,
		srv.aprod,
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
	ctx, span := srv.plat.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := rkcy.ExecuteTxnSync(
		ctx,
		srv.plat,
		srv.aprod,
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
	ctx, span := srv.plat.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := rkcy.ExecuteTxnSync(
		ctx,
		srv.plat,
		srv.aprod,
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
	ctx, span := srv.plat.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := rkcy.ExecuteTxnSync(
		ctx,
		srv.plat,
		srv.aprod,
		txn.ReadCharacter(req.Id),
		timeout(),
		srv.wg,
	)
	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &CharacterResponse{Character: resProto.Instance.(*pb.Character), Related: resProto.Related.(*pb.CharacterRelated)}, nil
}

func (srv server) CreateCharacter(ctx context.Context, plyr *pb.Character) (*pb.Character, error) {
	ctx, span := srv.plat.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := rkcy.ExecuteTxnSync(
		ctx,
		srv.plat,
		srv.aprod,
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
	ctx, span := srv.plat.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := rkcy.ExecuteTxnSync(
		ctx,
		srv.plat,
		srv.aprod,
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
	ctx, span := srv.plat.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := rkcy.ExecuteTxnSync(
		ctx,
		srv.plat,
		srv.aprod,
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
	ctx, span := srv.plat.Telem().StartFunc(ctx)
	defer span.End()

	resProto, err := rkcy.ExecuteTxnSync(
		ctx,
		srv.plat,
		srv.aprod,
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

func (srv server) ConductTrade(ctx context.Context, tr *pb.TradeRequest) (*rkcypb.Void, error) {
	ctx, span := srv.plat.Telem().StartFunc(ctx)
	defer span.End()

	_, err := rkcy.ExecuteTxnSync(
		ctx,
		srv.plat,
		srv.aprod,
		txn.Trade(tr),
		timeout(),
		srv.wg,
	)
	if err != nil {
		rkcy.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &rkcypb.Void{}, nil
}

func runServer(
	ctx context.Context,
	plat rkcy.Platform,
	aprod rkcy.ApecsProducer,
	wg *sync.WaitGroup,
) {
	srv := server{
		plat:  plat,
		aprod: aprod,
		wg:    wg,
	}
	rkcy.ServeGrpcGateway(ctx, srv)
}

func serve(plat rkcy.Platform) {
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

	aprod := plat.NewApecsProducer(
		ctx,
		&rkcypb.TopicTarget{
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

	go runServer(ctx, plat, aprod, &wg)

	select {
	case <-interruptCh:
		log.Info().
			Msg("edge server stopped")
		cancel()
		wg.Wait()
		return
	}
}
