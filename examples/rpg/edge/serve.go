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

	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	otel_codes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/version"

	"github.com/lachlanorr/rocketcycle/examples/rpg/commands"
	"github.com/lachlanorr/rocketcycle/examples/rpg/consts"
	"github.com/lachlanorr/rocketcycle/examples/rpg/storage"
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
	concernName string,
	command rkcy.Command,
	key string,
	payload *rkcy.Buffer,
) (*rkcy.ApecsTxn_Step_Result, error) {

	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	var steps []rkcy.Step
	if command == rkcy.Command_CREATE || command == rkcy.Command_UPDATE {
		var validateCmd rkcy.Command
		if command == rkcy.Command_CREATE {
			validateCmd = rkcy.Command_VALIDATE_NEW
		} else {
			validateCmd = rkcy.Command_VALIDATE_EXISTING
		}

		// CREATE/UPDATE get a validate step first
		steps = append(steps, rkcy.Step{
			ConcernName: concernName,
			Command:     validateCmd,
			Key:         key,
		})
	}
	steps = append(steps, rkcy.Step{
		ConcernName: concernName,
		Command:     command,
		Key:         key,
	})

	result, err := aprod.ExecuteTxnSync(
		ctx,
		false,
		payload,
		steps,
		time.Duration(settings.TimeoutSecs)*time.Second,
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
	command rkcy.Command,
	plyr *storage.Player,
) (*storage.Player, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	if command == rkcy.Command_CREATE {
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
	payload, err := storage.Marshal(int32(storage.ResourceType_PLAYER), plyr)

	result, err := processCrudRequest(ctx, traceId, consts.Player, command, plyr.Id, payload)
	if err != nil {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Failed to processCrudRequest")
		recordError(span, err)
		return nil, err
	}

	mdl, err := storage.Unmarshal(result.Payload)
	if err != nil {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Failed to Unmarshal Payload")
		recordError(span, err)
		return nil, status.New(codes.Internal, err.Error()).Err()
	}
	playerResult, ok := mdl.(*storage.Player)
	if !ok {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Unmarshal returned wrong type")
		recordError(span, err)
		return nil, status.New(codes.Internal, "Unmarshal returned wrong type").Err()
	}

	return playerResult, nil
}

func processCrudRequestCharacter(
	ctx context.Context,
	traceId string,
	command rkcy.Command,
	char *storage.Character,
) (*storage.Character, error) {

	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()

	if command == rkcy.Command_CREATE {
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
	payload, err := storage.Marshal(int32(storage.ResourceType_CHARACTER), char)

	result, err := processCrudRequest(ctx, traceId, consts.Character, command, char.Id, payload)
	if err != nil {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Failed to processCrudRequest")
		recordError(span, err)
		return nil, err
	}

	mdl, err := storage.Unmarshal(result.Payload)
	if err != nil {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Failed to Unmarshal Payload")
		recordError(span, err)
		return nil, status.New(codes.Internal, err.Error()).Err()
	}
	characterResult, ok := mdl.(*storage.Character)
	if !ok {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Unmarshal returned wrong type")
		recordError(span, err)
		return nil, status.New(codes.Internal, "Unmarshal returned wrong type").Err()
	}

	return characterResult, nil
}

func (server) ReadPlayer(ctx context.Context, req *RpgRequest) (*storage.Player, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	return processCrudRequestPlayer(ctx, traceId, rkcy.Command_READ, &storage.Player{Id: req.Id})
}

func (server) CreatePlayer(ctx context.Context, plyr *storage.Player) (*storage.Player, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	return processCrudRequestPlayer(ctx, traceId, rkcy.Command_CREATE, plyr)
}

func (server) UpdatePlayer(ctx context.Context, plyr *storage.Player) (*storage.Player, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	return processCrudRequestPlayer(ctx, traceId, rkcy.Command_UPDATE, plyr)
}

func (server) DeletePlayer(ctx context.Context, req *RpgRequest) (*RpgResponse, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	_, err := processCrudRequest(ctx, traceId, consts.Player, rkcy.Command_DELETE, req.Id, nil)
	if err != nil {
		return nil, err
	}
	return &RpgResponse{Id: req.Id}, nil
}

func (server) ReadCharacter(ctx context.Context, req *RpgRequest) (*storage.Character, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	return processCrudRequestCharacter(ctx, traceId, rkcy.Command_READ, &storage.Character{Id: req.Id})
}

func (server) CreateCharacter(ctx context.Context, char *storage.Character) (*storage.Character, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	return processCrudRequestCharacter(ctx, traceId, rkcy.Command_CREATE, char)
}

func (server) UpdateCharacter(ctx context.Context, char *storage.Character) (*storage.Character, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	return processCrudRequestCharacter(ctx, traceId, rkcy.Command_UPDATE, char)
}

func (server) DeleteCharacter(ctx context.Context, req *RpgRequest) (*RpgResponse, error) {
	ctx, traceId, span := rkcy.Telem().StartRequest(ctx)
	defer span.End()
	_, err := processCrudRequest(ctx, traceId, consts.Character, rkcy.Command_DELETE, req.Id, nil)
	if err != nil {
		return nil, err
	}
	return &RpgResponse{Id: req.Id}, nil
}

func (server) FundCharacter(ctx context.Context, fr *storage.FundingRequest) (*storage.Character, error) {
	ctx, span := rkcy.Telem().StartFunc(ctx)
	defer span.End()
	traceId := span.SpanContext().TraceID().String()

	payload, err := storage.Marshal(int32(storage.ResourceType_FUNDING_REQUEST), fr)
	if err != nil {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Failed to Marshal Payload")
		recordError(span, err)
		return nil, status.New(codes.Internal, err.Error()).Err()
	}

	result, err := aprod.ExecuteTxnSync(
		ctx,
		false,
		payload,
		[]rkcy.Step{
			{
				ConcernName: consts.Character,
				Command:     commands.Command_FUND,
				Key:         fr.CharacterId,
			},
		},
		time.Duration(settings.TimeoutSecs)*time.Second,
	)
	if err != nil {
		recordError(span, err)
		return nil, status.New(codes.Internal, err.Error()).Err()
	}

	mdl, err := storage.Unmarshal(result.Payload)
	if err != nil {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Failed to Unmarshal Payload")
		recordError(span, err)
		return nil, status.New(codes.Internal, err.Error()).Err()
	}
	characterResult, ok := mdl.(*storage.Character)
	if !ok {
		log.Error().
			Err(err).
			Str("TraceId", traceId).
			Msg("Unmarshal returned wrong type")
		recordError(span, err)
		return nil, status.New(codes.Internal, "Unmarshal returned wrong type").Err()
	}
	return characterResult, nil
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

	aprod = rkcy.NewApecsProducer(
		ctx,
		settings.BootstrapServers,
		rkcy.PlatformName(),
		&rkcy.TopicTarget{
			BootstrapServers: settings.BootstrapServers,
			TopicName:        settings.Topic,
			Partition:        settings.Partition,
		},
	)

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
