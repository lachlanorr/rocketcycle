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
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lachlanorr/rocketcycle/pkg/apecs"
	"github.com/lachlanorr/rocketcycle/pkg/platform"
	"github.com/lachlanorr/rocketcycle/pkg/portal"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/pkg/telem"
	"github.com/lachlanorr/rocketcycle/version"

	"github.com/lachlanorr/rocketcycle/examples/rpg/logic/txn"
	"github.com/lachlanorr/rocketcycle/examples/rpg/pb"
)

//go:embed static/docs
var docsFiles embed.FS

type server struct {
	UnimplementedRpgServiceServer

	plat *platform.Platform
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
	ctx, span := telem.StartFunc(ctx)
	defer span.End()

	resProto, err := apecs.ExecuteTxnSync(
		ctx,
		srv.plat,
		txn.ReadPlayer(req.Id),
		timeout(),
	)
	if err != nil {
		telem.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &PlayerResponse{Player: resProto.Instance.(*pb.Player), Related: resProto.Related.(*pb.PlayerRelated)}, nil
}

func (srv server) CreatePlayer(ctx context.Context, plyr *pb.Player) (*pb.Player, error) {
	ctx, span := telem.StartFunc(ctx)
	defer span.End()

	resProto, err := apecs.ExecuteTxnSync(
		ctx,
		srv.plat,
		txn.CreatePlayer(plyr),
		timeout(),
	)
	if err != nil {
		telem.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resProto.Instance.(*pb.Player), nil
}

func (srv server) UpdatePlayer(ctx context.Context, plyr *pb.Player) (*pb.Player, error) {
	ctx, span := telem.StartFunc(ctx)
	defer span.End()

	resProto, err := apecs.ExecuteTxnSync(
		ctx,
		srv.plat,
		txn.UpdatePlayer(plyr),
		timeout(),
	)
	if err != nil {
		telem.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resProto.Instance.(*pb.Player), nil
}

func (srv server) DeletePlayer(ctx context.Context, req *RpgRequest) (*pb.Player, error) {
	ctx, span := telem.StartFunc(ctx)
	defer span.End()

	resProto, err := apecs.ExecuteTxnSync(
		ctx,
		srv.plat,
		txn.DeletePlayer(req.Id),
		timeout(),
	)
	if err != nil {
		telem.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resProto.Instance.(*pb.Player), nil
}

func (srv server) ReadCharacter(ctx context.Context, req *RpgRequest) (*CharacterResponse, error) {
	ctx, span := telem.StartFunc(ctx)
	defer span.End()

	resProto, err := apecs.ExecuteTxnSync(
		ctx,
		srv.plat,
		txn.ReadCharacter(req.Id),
		timeout(),
	)
	if err != nil {
		telem.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &CharacterResponse{Character: resProto.Instance.(*pb.Character), Related: resProto.Related.(*pb.CharacterRelated)}, nil
}

func (srv server) CreateCharacter(ctx context.Context, plyr *pb.Character) (*pb.Character, error) {
	ctx, span := telem.StartFunc(ctx)
	defer span.End()

	resProto, err := apecs.ExecuteTxnSync(
		ctx,
		srv.plat,
		txn.CreateCharacter(plyr),
		timeout(),
	)
	if err != nil {
		telem.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resProto.Instance.(*pb.Character), nil
}

func (srv server) UpdateCharacter(ctx context.Context, plyr *pb.Character) (*pb.Character, error) {
	ctx, span := telem.StartFunc(ctx)
	defer span.End()

	resProto, err := apecs.ExecuteTxnSync(
		ctx,
		srv.plat,
		txn.UpdateCharacter(plyr),
		timeout(),
	)
	if err != nil {
		telem.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resProto.Instance.(*pb.Character), nil
}

func (srv server) DeleteCharacter(ctx context.Context, req *RpgRequest) (*pb.Character, error) {
	ctx, span := telem.StartFunc(ctx)
	defer span.End()

	resProto, err := apecs.ExecuteTxnSync(
		ctx,
		srv.plat,
		txn.DeleteCharacter(req.Id),
		timeout(),
	)
	if err != nil {
		telem.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resProto.Instance.(*pb.Character), nil
}

func (srv server) FundCharacter(ctx context.Context, fr *pb.FundingRequest) (*pb.Character, error) {
	ctx, span := telem.StartFunc(ctx)
	defer span.End()

	resProto, err := apecs.ExecuteTxnSync(
		ctx,
		srv.plat,
		txn.Fund(fr),
		timeout(),
	)
	if err != nil {
		telem.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resProto.Instance.(*pb.Character), nil
}

func (srv server) ConductTrade(ctx context.Context, tr *pb.TradeRequest) (*rkcypb.Void, error) {
	ctx, span := telem.StartFunc(ctx)
	defer span.End()

	_, err := apecs.ExecuteTxnSync(
		ctx,
		srv.plat,
		txn.Trade(tr),
		timeout(),
	)
	if err != nil {
		telem.RecordSpanError(span, err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &rkcypb.Void{}, nil
}

func runServer(
	ctx context.Context,
	plat *platform.Platform,
) {
	srv := server{
		plat: plat,
	}
	portal.ServeGrpcGateway(ctx, srv)
}

func serve(plat *platform.Platform) {
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

	go runServer(ctx, plat)

	select {
	case <-interruptCh:
		log.Info().
			Msg("edge server stopped")
		cancel()
		plat.WaitGroup().Wait()
		return
	}
}
