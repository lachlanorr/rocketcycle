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

func CodeTranslate(code rkcy_pb.Code) codes.Code {
	switch code {
	case rkcy_pb.Code_OK:
		return codes.OK
	case rkcy_pb.Code_NOT_FOUND:
		return codes.NotFound
	case rkcy_pb.Code_FAILED_CONSTRAINT:
		return codes.AlreadyExists

	case rkcy_pb.Code_INTERNAL:
		fallthrough
	case rkcy_pb.Code_MARSHAL_FAILED:
		fallthrough
	case rkcy_pb.Code_CONNECTION:
		fallthrough
	case rkcy_pb.Code_UNKNOWN_COMMAND:
		fallthrough
	default:
		return codes.Internal
	}
}

type RespChan struct {
	ReqId  string
	RespCh chan *rkcy_pb.ApecsTxn
}

var registerCh chan *RespChan = make(chan *RespChan, 10)

func consumeResponseTopic(ctx context.Context, bootstrapServers string, fullTopic string, partition int32) {
	reqMap := make(map[string]chan *rkcy_pb.ApecsTxn)

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

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("watchTopic.consume exiting, ctx.Done()")
			return
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
					reqMap[rch.ReqId] = rch.RespCh
					log.Info().Msgf("reqMap: %+v", reqMap)
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
				if directive == rkcy_pb.Directive_APECS_TXN {
					reqId := rkcy.GetReqId(msg)
					txnCh, ok := reqMap[reqId]
					if !ok {
						log.Error().
							Str("ReqId", reqId).
							Msg("ReqId not found in reqMap")
					} else {
						delete(reqMap, reqId)
						txn := rkcy_pb.ApecsTxn{}
						err := proto.Unmarshal(msg.Value, &txn)
						if err != nil {
							log.Error().
								Err(err).
								Str("ReqId", reqId).
								Msg("Failed to Unmarshal ApecsTxn")
						} else {
							txnCh <- &txn
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

func waitForResponse(ctx context.Context, rspCh <-chan *rkcy_pb.ApecsTxn, timeoutSecs int) (*rkcy_pb.ApecsTxn, error) {
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

	reqId := uuid.NewString()
	respChan := RespChan{
		ReqId:  reqId,
		RespCh: make(chan *rkcy_pb.ApecsTxn),
	}
	registerCh <- &respChan

	err = aprod.ExecuteTxn(
		reqId,
		&rkcy_pb.ResponseTarget{
			BootstrapServers: settings.BootstrapServers,
			TopicName:        settings.Topic,
			Partition:        settings.Partition,
		},
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
			Str("ReqId", reqId).
			Msg("failed to ExecuteTxn")
		return nil, status.New(codes.Internal, "failed to ExecuteTxn").Err()
	}

	log.Info().
		Str("ReqId", reqId).
		Msg("CreatePlayer")

	txnRsp, err := waitForResponse(ctx, respChan.RespCh, consts.DefaultTimeoutSecs)
	if err != nil {
		log.Error().
			Err(err).
			Str("ReqId", reqId).
			Msg("failed to waitForResponse")
		return nil, status.New(codes.Internal, err.Error()).Err()
	}

	success, result := rkcy.ApecsTxnResult(txnRsp)
	if success {
		playerResult := rpg_pb.Player{}
		err = proto.Unmarshal(result.Payload, &playerResult)
		if err != nil {
			log.Error().
				Err(err).
				Str("ReqId", reqId).
				Msg("Failed to Unmarshal Payload")
			return nil, status.New(codes.Internal, err.Error()).Err()
		}

		return &playerResult, nil
	} else {
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
