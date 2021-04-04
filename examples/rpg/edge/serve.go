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
	spb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

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

func CodeTranslate(code rkcy.Code) codes.Code {
	switch code {
	case rkcy.Code_OK:
		return codes.OK
	case rkcy.Code_NOT_FOUND:
		return codes.NotFound
	case rkcy.Code_CONSTRAINT_VIOLATION:
		return codes.AlreadyExists

	case rkcy.Code_INVALID_ARGUMENT:
		return codes.InvalidArgument

	case rkcy.Code_INTERNAL:
		fallthrough
	case rkcy.Code_MARSHAL_FAILED:
		fallthrough
	case rkcy.Code_CONNECTION:
		fallthrough
	case rkcy.Code_UNKNOWN_COMMAND:
		fallthrough
	default:
		return codes.Internal
	}
}

type RespChan struct {
	ReqId     string
	RespCh    chan *rkcy.ApecsTxn
	StartTime time.Time
}

var registerCh chan *RespChan = make(chan *RespChan, 10)

func consumeResponseTopic(ctx context.Context, bootstrapServers string, fullTopic string, partition int32) {
	reqMap := make(map[string]*RespChan)

	groupName := fmt.Sprintf("rkcy_%s_edge__%s_%d", rkcy.PlatformName(), fullTopic, partition)

	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        bootstrapServers,
		"group.id":                 groupName,
		"enable.auto.commit":       true, // librdkafka will commit to brokers for us on an interval and when we close consumer
		"enable.auto.offset.store": true, // librdkafka will commit to local store to get "at most once" behavior
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Str("BoostrapServers", bootstrapServers).
			Str("GroupId", groupName).
			Msg("Unable to kafka.NewConsumer")
	}
	defer cons.Close()

	err = cons.Assign([]kafka.TopicPartition{
		{
			Topic:     &fullTopic,
			Partition: partition,
			Offset:    kafka.OffsetStored,
		},
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to Assign")
	}

	cleanupTicker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("watchTopic.consume exiting, ctx.Done()")
			return
		case <-cleanupTicker.C:
			for reqId, rspCh := range reqMap {
				now := time.Now()
				if now.Sub(rspCh.StartTime) >= time.Second*10 {
					log.Warn().
						Str("ReqId", reqId).
						Msgf("Deleteing request channel info, this is not normal and this transaction may have been lost")
					rspCh.RespCh <- nil
					delete(reqMap, reqId)
				}
			}
		default:
			msg, err := cons.ReadMessage(time.Millisecond * 10)

			// Read all registration messages here so we are sure to catch them before handling message
			moreRegistrations := true
			for moreRegistrations {
				select {
				case rch := <-registerCh:
					_, ok := reqMap[rch.ReqId]
					if ok {
						log.Error().
							Str("ReqId", rch.ReqId).
							Msg("ReqId already registered for responses, replacing with new value")
					}
					reqMap[rch.ReqId] = rch
				default:
					moreRegistrations = false
				}
			}

			timedOut := err != nil && err.(kafka.Error).Code() == kafka.ErrTimedOut
			if err != nil && !timedOut {
				log.Error().
					Err(err).
					Msg("Error during ReadMessage")
			} else if !timedOut && msg != nil {
				directive := rkcy.GetDirective(msg)
				if directive == rkcy.Directive_APECS_TXN {
					reqId := rkcy.GetReqId(msg)
					rspCh, ok := reqMap[reqId]
					if !ok {
						log.Error().
							Str("ReqId", reqId).
							Msg("ReqId not found in reqMap")
					} else {
						delete(reqMap, reqId)
						txn := rkcy.ApecsTxn{}
						err := proto.Unmarshal(msg.Value, &txn)
						if err != nil {
							log.Error().
								Err(err).
								Str("ReqId", reqId).
								Msg("Failed to Unmarshal ApecsTxn")
						} else {
							rspCh.RespCh <- &txn
						}
					}
				} else {
					log.Warn().
						Int("Directive", int(directive)).
						Msg("Invalid directive on ApecsTxn topic")
				}
			}
		}
	}
}

func waitForResponse(ctx context.Context, rspCh <-chan *rkcy.ApecsTxn, timeoutSecs int) (*rkcy.ApecsTxn, error) {
	timer := time.NewTimer(time.Duration(timeoutSecs) * time.Second)

	select {
	case <-ctx.Done():
		log.Info().
			Msg("edge server request exiting, ctx.Done()")
		return nil, status.New(codes.Internal, "context closed").Err()
	case <-timer.C:
		return nil, status.New(codes.DeadlineExceeded, "time out waiting on response").Err()
	case txn := <-rspCh:
		return txn, nil
	}
}

func processCrudRequest(
	ctx context.Context,
	concernName string,
	command rkcy.Command,
	key string,
	payload *rkcy.Buffer,
) (string, *rkcy.ApecsTxn_Step_Result, error) {

	reqId := uuid.NewString()

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

	respChan := RespChan{
		ReqId:     reqId,
		RespCh:    make(chan *rkcy.ApecsTxn),
		StartTime: time.Now(),
	}
	registerCh <- &respChan

	err := aprod.ExecuteTxn(
		reqId,
		&rkcy.ResponseTarget{
			BootstrapServers: settings.BootstrapServers,
			TopicName:        settings.Topic,
			Partition:        settings.Partition,
		},
		false,
		payload,
		steps,
	)

	if err != nil {
		return reqId, nil, status.New(codes.Internal, "failed to ExecuteTxn").Err()
	}

	txnRsp, err := waitForResponse(ctx, respChan.RespCh, settings.TimeoutSecs)
	if err != nil {
		return reqId, nil, status.New(codes.Internal, err.Error()).Err()
	}
	if txnRsp == nil {
		return reqId, nil, status.New(codes.Internal, "nil txn received").Err()
	}

	success, result := rkcy.ApecsTxnResult(txnRsp)
	if !success {
		details := make([]*anypb.Any, 0, 1)
		resultAny, err := anypb.New(result)
		if err != nil {
			log.Error().
				Err(err).
				Str("ReqId", reqId).
				Msg("Unable to convert result to Any")
		} else {
			details = append(details, resultAny)
		}
		stat := spb.Status{
			Code:    int32(CodeTranslate(result.Code)),
			Message: "failure",
			Details: details,
		}
		return reqId, nil, status.ErrorProto(&stat)
	}
	return reqId, result, nil
}

func processCrudRequestPlayer(
	ctx context.Context,
	command rkcy.Command,
	plyr *storage.Player,
) (*storage.Player, error) {

	if command == rkcy.Command_CREATE {
		if plyr.Id != "" {
			return nil, status.New(codes.InvalidArgument, "non empty 'id' field in payload").Err()
		}
		plyr.Id = uuid.NewString()
	} else {
		if plyr.Id == "" {
			return nil, status.New(codes.InvalidArgument, "empty 'id' field in payload").Err()
		}
	}
	payload, err := storage.Marshal(int32(storage.ResourceType_PLAYER), plyr)

	reqId, result, err := processCrudRequest(ctx, consts.Player, command, plyr.Id, payload)
	if err != nil {
		log.Error().
			Err(err).
			Str("ReqId", reqId).
			Msg("Failed to processCrudRequest")
		return nil, err
	}

	mdl, err := storage.Unmarshal(result.Payload)
	if err != nil {
		log.Error().
			Err(err).
			Str("ReqId", reqId).
			Msg("Failed to Unmarshal Payload")
		return nil, status.New(codes.Internal, err.Error()).Err()
	}
	playerResult, ok := mdl.(*storage.Player)
	if !ok {
		log.Error().
			Err(err).
			Str("ReqId", reqId).
			Msg("Unmarshal returned wrong type")
		return nil, status.New(codes.Internal, "Unmarshal returned wrong type").Err()
	}

	return playerResult, nil
}

func processCrudRequestCharacter(
	ctx context.Context,
	command rkcy.Command,
	char *storage.Character,
) (*storage.Character, error) {

	if command == rkcy.Command_CREATE {
		if char.Id != "" {
			return nil, status.New(codes.InvalidArgument, "non empty 'id' field in payload").Err()
		}
		char.Id = uuid.NewString()
	} else {
		if char.Id == "" {
			return nil, status.New(codes.InvalidArgument, "empty 'id' field in payload").Err()
		}
	}
	payload, err := storage.Marshal(int32(storage.ResourceType_CHARACTER), char)

	reqId, result, err := processCrudRequest(ctx, consts.Character, command, char.Id, payload)
	if err != nil {
		log.Error().
			Err(err).
			Str("ReqId", reqId).
			Msg("Failed to processCrudRequest")
		return nil, err
	}

	mdl, err := storage.Unmarshal(result.Payload)
	if err != nil {
		log.Error().
			Err(err).
			Str("ReqId", reqId).
			Msg("Failed to Unmarshal Payload")
		return nil, status.New(codes.Internal, err.Error()).Err()
	}
	characterResult, ok := mdl.(*storage.Character)
	if !ok {
		log.Error().
			Err(err).
			Str("ReqId", reqId).
			Msg("Unmarshal returned wrong type")
		return nil, status.New(codes.Internal, "Unmarshal returned wrong type").Err()
	}

	return characterResult, nil
}

func (server) ReadPlayer(ctx context.Context, req *RpgRequest) (*storage.Player, error) {
	return processCrudRequestPlayer(ctx, rkcy.Command_READ, &storage.Player{Id: req.Id})
}

func (server) CreatePlayer(ctx context.Context, plyr *storage.Player) (*storage.Player, error) {
	return processCrudRequestPlayer(ctx, rkcy.Command_CREATE, plyr)
}

func (server) UpdatePlayer(ctx context.Context, plyr *storage.Player) (*storage.Player, error) {
	return processCrudRequestPlayer(ctx, rkcy.Command_UPDATE, plyr)
}

func (server) DeletePlayer(ctx context.Context, req *RpgRequest) (*RpgResponse, error) {
	_, _, err := processCrudRequest(ctx, consts.Player, rkcy.Command_DELETE, req.Id, nil)
	if err != nil {
		return nil, err
	}
	return &RpgResponse{Id: req.Id}, nil
}

func (server) ReadCharacter(ctx context.Context, req *RpgRequest) (*storage.Character, error) {
	return processCrudRequestCharacter(ctx, rkcy.Command_READ, &storage.Character{Id: req.Id})
}

func (server) CreateCharacter(ctx context.Context, char *storage.Character) (*storage.Character, error) {
	return processCrudRequestCharacter(ctx, rkcy.Command_CREATE, char)
}

func (server) UpdateCharacter(ctx context.Context, char *storage.Character) (*storage.Character, error) {
	return processCrudRequestCharacter(ctx, rkcy.Command_UPDATE, char)
}

func (server) DeleteCharacter(ctx context.Context, req *RpgRequest) (*RpgResponse, error) {
	_, _, err := processCrudRequest(ctx, consts.Character, rkcy.Command_DELETE, req.Id, nil)
	if err != nil {
		return nil, err
	}
	return &RpgResponse{Id: req.Id}, nil
}

func (server) FundCharacter(ctx context.Context, fr *storage.FundingRequest) (*storage.Character, error) {
	reqId := uuid.NewString()
	payload, err := storage.Marshal(int32(storage.ResourceType_FUNDING_REQUEST), fr)
	if err != nil {
		log.Error().
			Err(err).
			Str("ReqId", reqId).
			Msg("Failed to Marshal Payload")
		return nil, status.New(codes.Internal, err.Error()).Err()
	}

	respChan := RespChan{
		ReqId:     reqId,
		RespCh:    make(chan *rkcy.ApecsTxn),
		StartTime: time.Now(),
	}
	registerCh <- &respChan

	err = aprod.ExecuteTxn(
		reqId,
		&rkcy.ResponseTarget{
			BootstrapServers: settings.BootstrapServers,
			TopicName:        settings.Topic,
			Partition:        settings.Partition,
		},
		false,
		payload,
		[]rkcy.Step{
			{
				ConcernName: consts.Character,
				Command:     commands.Command_FUND,
				Key:         fr.CharacterId,
			},
		},
	)
	if err != nil {
		return nil, status.New(codes.Internal, "failed to ExecuteTxn").Err()
	}

	txnRsp, err := waitForResponse(ctx, respChan.RespCh, settings.TimeoutSecs)
	if err != nil {
		return nil, status.New(codes.Internal, err.Error()).Err()
	}
	if txnRsp == nil {
		return nil, status.New(codes.Internal, "nil txn received").Err()
	}

	success, result := rkcy.ApecsTxnResult(txnRsp)
	if !success {
		details := make([]*anypb.Any, 0, 1)
		resultAny, err := anypb.New(result)
		if err != nil {
			log.Error().
				Err(err).
				Str("ReqId", reqId).
				Msg("Unable to convert result to Any")
		} else {
			details = append(details, resultAny)
		}
		stat := spb.Status{
			Code:    int32(CodeTranslate(result.Code)),
			Message: "failure",
			Details: details,
		}
		return nil, status.ErrorProto(&stat)
	}

	mdl, err := storage.Unmarshal(result.Payload)
	if err != nil {
		log.Error().
			Err(err).
			Str("ReqId", reqId).
			Msg("Failed to Unmarshal Payload")
		return nil, status.New(codes.Internal, err.Error()).Err()
	}
	characterResult, ok := mdl.(*storage.Character)
	if !ok {
		log.Error().
			Err(err).
			Str("ReqId", reqId).
			Msg("Unmarshal returned wrong type")
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

	aprod = rkcy.NewApecsProducer(ctx, settings.BootstrapServers, rkcy.PlatformName())
	if aprod == nil {
		log.Fatal().
			Msg("Failed to NewApecsProducer")
	}

	go consumeResponseTopic(ctx, settings.BootstrapServers, settings.Topic, settings.Partition)
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
