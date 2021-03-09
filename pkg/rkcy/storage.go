// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rkcy/pkg/rkcy/pb"
	"github.com/lachlanorr/rkcy/version"
)

func processStorageRequest(req *pb.ApecsStorageRequest, mro *pb.MostRecentOffset) *pb.ApecsStorageResult {
	hndlr, ok := platformImpl.StorageHandlers[req.ConcernName]
	if !ok {
		log.Error().
			Msg("processStorageRequest: Unknown Concern: " + req.ConcernName)
		return nil
	}

	var res *pb.ApecsStorageResult

	switch req.Op {
	case pb.ApecsStorageRequest_GET:
		log.Error().
			Msg("processStorageRequest: GET not implemented")
	case pb.ApecsStorageRequest_CREATE:
		res = hndlr.Create(req.Uid, req.Key, req.Payload, mro)
		log.Info().
			Msg("CREATE processed")
	case pb.ApecsStorageRequest_UPDATE:
		log.Error().
			Msg("processStorageRequest: UPDATE not implemented")
	case pb.ApecsStorageRequest_DELETE:
		log.Error().
			Msg("processStorageRequest: DELETE not implemented")
	}

	return res
}

func calcMro(fullTopic string, offset int64) (*pb.MostRecentOffset, error) {
	parts := strings.Split(fullTopic, ".")
	gen, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return nil, err
	}

	return &pb.MostRecentOffset{
		Generation: int32(gen),
		Offset:     offset,
	}, nil
}

func consumeStorage(ctx context.Context, clusterBoostrap string, fullTopic string, partition int32) {
	groupName := fmt.Sprintf("rkcy_%s", fullTopic)
	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        clusterBoostrap,
		"group.id":                 groupName,
		"enable.auto.commit":       true,  // librdkafka will commit to brokers for us on an interval and when we close consumer
		"enable.auto.offset.store": false, // we explicitely commit to local store to get "at least once" behavior
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("failed to kafka.NewConsumer")
		return
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
		return
	}

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Msg("consumeStorage exiting, ctx.Done()")
			return
		default:
			msg, err := cons.ReadMessage(time.Second * 5)
			timedOut := err != nil && err.(kafka.Error).Code() == kafka.ErrTimedOut
			if err != nil && !timedOut {
				log.Error().
					Err(err).
					Msg("Error during ReadMessage")
				return
			}
			if !timedOut && msg != nil {
				directive := getDirective(msg)
				mro, err := calcMro(fullTopic, int64(msg.TopicPartition.Offset))
				if err != nil {
					log.Error().
						Err(err).
						Msg("Invalid generation: " + fullTopic)
					return
				}

				if directive == pb.Directive_APECS_STORAGE_REQUEST {
					req := pb.ApecsStorageRequest{}
					err := proto.Unmarshal(msg.Value, &req)
					if err != nil {
						log.Error().
							Err(err).
							Msg("Failed to Unmarshal ApecsStorageRequest")
					} else {
						res := processStorageRequest(&req, mro)

						if res != nil {
							_ /*resSer*/, err := proto.Marshal(res)
							if err != nil {
								log.Error().
									Err(err).
									Msg("failed to marshal ApecsStorageResult")
							} else {

							}
						}
					}
				}

				_, err = cons.StoreOffsets([]kafka.TopicPartition{
					{
						Topic:     &fullTopic,
						Partition: partition,
						Offset:    msg.TopicPartition.Offset + 1,
					},
				})
				if err != nil {
					log.Error().
						Err(err).
						Msgf("Unable to store offsets %s/%d/%d", fullTopic, partition, msg.TopicPartition.Offset)
					return
				}
			}
		}
	}

}

func cobraStorage(cmd *cobra.Command, args []string) {
	log.Info().
		Str("GitCommit", version.GitCommit).
		Msg("APECS storage consumer started")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go consumeStorage(ctx, settings.BootstrapServers, settings.Topic, settings.Partition)

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt)
	select {
	case <-interruptCh:
		log.Info().
			Msg("APECS storage consumer stopped")
		cancel()
		return
	}

	/*


		conn, err := pgx.Connect(context.Background(), "postgresql://postgres@127.0.0.1:5432/rpg")
		if err != nil {
			log.Fatal().
				Err(err).
				Msg("Unable to connect to database")
		}
		defer conn.Close(context.Background())

		var username string
		var active bool
		err = conn.QueryRow(context.Background(), "select username, active from rpg.player where id=$1", 1).Scan(&username, &active)
		if err != nil {
			log.Fatal().
				Err(err).
				Msg("QueryRow failed")
		}

		log.Info().Msg(fmt.Sprintf("username = %s, active = %t", username, active))

		time.Sleep(30 * time.Second)

		log.Info().Msg("rcstore exit")
	*/
}
