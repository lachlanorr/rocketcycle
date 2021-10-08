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

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	otel_codes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/version"

	"github.com/lachlanorr/rocketcycle/examples/rpg/concerns"
	"github.com/lachlanorr/rocketcycle/examples/rpg/consts"
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

func processCrudRequest(
	ctx context.Context,
	traceId string,
	concern string,
	command string,
	key string,
	payload proto.Message,
	wg *sync.WaitGroup,
) (*rkcy.ApecsTxn_Step_Result, error) {

	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	var steps []rkcy.Step
	if command == rkcy.CREATE || command == rkcy.UPDATE {
		var validateCmd string
		if command == rkcy.CREATE {
			validateCmd = rkcy.VALIDATE_CREATE
		} else {
			validateCmd = rkcy.VALIDATE_UPDATE
		}

		// CREATE/UPDATE get a validate step first
		steps = append(steps, rkcy.Step{
			Concern: concern,
			Command: validateCmd,
			Key:     key,
		})
	}
	steps = append(steps, rkcy.Step{
		Concern: concern,
		Command: command,
		Key:     key,
	})

	result, err := aprod.ExecuteTxnSync(
		ctx,
		false,
		payload,
		steps,
		time.Duration(settings.TimeoutSecs)*time.Second,
		wg,
	)
	if err != nil {
		recordError(span, err)
		return nil, status.New(codes.Internal, err.Error()).Err()
	}

	return result, nil
}

func recordError(span trace.Span, err error) {
	span.SetStatus(otel_codes.Error, err.Error())
}

func processCrudRequestPlayer(
	ctx context.Context,
	traceId string,
	command string,
	plyr *concerns.Player,
	wg *sync.WaitGroup,
) (*concerns.Player, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	if command == rkcy.CREATE {
		if plyr.Id != "" {
			err := status.New(codes.InvalidArgument, "non empty 'id' field in payload").Err()
			recordError(span, err)
			return nil, err
		}
		plyr.Id = uuid.NewString()
	} else {
		if plyr.Id == "" {
			err := status.New(codes.InvalidArgument, "non empty 'id' field in payload").Err()
			recordError(span, err)
			return nil, err
		}
	}

	result, err := processCrudRequest(ctx, traceId, consts.Player, command, plyr.Id, plyr, wg)
	if err != nil {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Failed to processCrudRequest")
		recordError(span, err)
		return nil, err
	}

	playerResult := &concerns.Player{}
	err = proto.Unmarshal(result.Payload, playerResult)
	if err != nil {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Failed to Unmarshal Payload")
		recordError(span, err)
		return nil, status.New(codes.Internal, err.Error()).Err()
	}

	return playerResult, nil
}

func processCrudRequestCharacter(
	ctx context.Context,
	traceId string,
	command string,
	char *concerns.Character,
	wg *sync.WaitGroup,
) (*concerns.Character, error) {

	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	if command == rkcy.CREATE {
		if char.Id != "" {
			err := status.New(codes.InvalidArgument, "non empty 'id' field in payload").Err()
			recordError(span, err)
			return nil, err
		}
		char.Id = uuid.NewString()
	} else {
		if char.Id == "" {
			err := status.New(codes.InvalidArgument, "empty 'id' field in payload").Err()
			recordError(span, err)
			return nil, err
		}
	}

	result, err := processCrudRequest(ctx, traceId, consts.Character, command, char.Id, char, wg)
	if err != nil {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Failed to processCrudRequest")
		recordError(span, err)
		return nil, err
	}

	characterResult := &concerns.Character{}
	err = proto.Unmarshal(result.Payload, characterResult)
	if err != nil {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Failed to Unmarshal Payload")
		recordError(span, err)
		return nil, status.New(codes.Internal, err.Error()).Err()
	}

	return characterResult, nil
}

func (srv server) ReadPlayer(ctx context.Context, req *RpgRequest) (*concerns.Player, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	return processCrudRequestPlayer(ctx, traceId, rkcy.READ, &concerns.Player{Id: req.Id}, srv.wg)
}

func (srv server) CreatePlayer(ctx context.Context, plyr *concerns.Player) (*concerns.Player, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	return processCrudRequestPlayer(ctx, traceId, rkcy.CREATE, plyr, srv.wg)
}

func (srv server) UpdatePlayer(ctx context.Context, plyr *concerns.Player) (*concerns.Player, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	return processCrudRequestPlayer(ctx, traceId, rkcy.UPDATE, plyr, srv.wg)
}

func (srv server) DeletePlayer(ctx context.Context, req *RpgRequest) (*RpgResponse, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	_, err := processCrudRequest(ctx, traceId, consts.Player, rkcy.DELETE, req.Id, nil, srv.wg)
	if err != nil {
		return nil, err
	}
	return &RpgResponse{Id: req.Id}, nil
}

func (srv server) ReadCharacter(ctx context.Context, req *RpgRequest) (*concerns.Character, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	return processCrudRequestCharacter(ctx, traceId, rkcy.READ, &concerns.Character{Id: req.Id}, srv.wg)
}

func (srv server) CreateCharacter(ctx context.Context, char *concerns.Character) (*concerns.Character, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	return processCrudRequestCharacter(ctx, traceId, rkcy.CREATE, char, srv.wg)
}

func (srv server) UpdateCharacter(ctx context.Context, char *concerns.Character) (*concerns.Character, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	return processCrudRequestCharacter(ctx, traceId, rkcy.UPDATE, char, srv.wg)
}

func (srv server) DeleteCharacter(ctx context.Context, req *RpgRequest) (*RpgResponse, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	_, err := processCrudRequest(ctx, traceId, consts.Character, rkcy.DELETE, req.Id, nil, srv.wg)
	if err != nil {
		return nil, err
	}
	return &RpgResponse{Id: req.Id}, nil
}

func (srv server) FundCharacter(ctx context.Context, fr *concerns.FundingRequest) (*concerns.Character, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()
	traceId := span.SpanContext().TraceID().String()

	result, err := aprod.ExecuteTxnSync(
		ctx,
		false,
		fr,
		[]rkcy.Step{
			{
				Concern: consts.Character,
				Command: "Fund",
				Key:     fr.CharacterId,
			},
		},
		time.Duration(settings.TimeoutSecs)*time.Second,
		srv.wg,
	)
	if err != nil {
		recordError(span, err)
		return nil, status.New(codes.Internal, err.Error()).Err()
	}

	characterResult := &concerns.Character{}
	err = proto.Unmarshal(result.Payload, characterResult)
	if err != nil {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Failed to Unmarshal Payload")
		recordError(span, err)
		return nil, status.New(codes.Internal, err.Error()).Err()
	}
	return characterResult, nil
}

func (srv server) ConductTrade(ctx context.Context, tr *TradeRequest) (*rkcy.Void, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()
	traceId := span.SpanContext().TraceID().String()

	result, err := aprod.ExecuteTxnSync(
		ctx,
		true,
		nil,
		[]rkcy.Step{
			{
				Concern: "Character",
				Command: "DebitFunds",
				Key:     tr.Lhs.CharacterId,
				Payload: tr.Lhs,
			},
			{
				Concern: "Character",
				Command: "DebitFunds",
				Key:     tr.Rhs.CharacterId,
				Payload: tr.Rhs,
			},
			{
				Concern: "Character",
				Command: "CreditFunds",
				Key:     tr.Lhs.CharacterId,
				Payload: tr.Rhs,
			},
			{
				Concern: "Character",
				Command: "CreditFunds",
				Key:     tr.Rhs.CharacterId,
				Payload: tr.Lhs,
			},
		},
		time.Duration(settings.TimeoutSecs)*time.Second,
		srv.wg,
	)

	if err != nil {
		recordError(span, err)
		return nil, status.New(codes.Internal, err.Error()).Err()
	}

	characterResult := &concerns.Character{}
	err = proto.Unmarshal(result.Payload, characterResult)
	if err != nil {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Failed to Unmarshal Payload")
		recordError(span, err)
		return nil, status.New(codes.Internal, err.Error()).Err()
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
