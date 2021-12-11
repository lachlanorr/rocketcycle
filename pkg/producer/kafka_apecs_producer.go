// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package producer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type ApecsKafkaProducer struct {
	ctx               context.Context
	plat              rkcy.Platform
	respTarget        *rkcypb.TopicTarget
	producers         map[string]map[rkcy.StandardTopicName]rkcy.Producer
	producersMtx      sync.Mutex
	respRegisterCh    chan *rkcy.RespChan
	respConsumerClose context.CancelFunc
}

func (akprod *ApecsKafkaProducer) Platform() rkcy.Platform {
	return akprod.plat
}

func (akprod *ApecsKafkaProducer) ResponseTarget() *rkcypb.TopicTarget {
	return akprod.respTarget
}

func (akprod *ApecsKafkaProducer) RegisterResponseChannel(respCh *rkcy.RespChan) {
	akprod.respRegisterCh <- respCh
}

func NewApecsKafkaProducer(
	ctx context.Context,
	plat rkcy.Platform,
	respTarget *rkcypb.TopicTarget,
	wg *sync.WaitGroup,
) *ApecsKafkaProducer {
	akprod := &ApecsKafkaProducer{
		ctx:        ctx,
		plat:       plat,
		respTarget: respTarget,

		producers:      make(map[string]map[rkcy.StandardTopicName]rkcy.Producer),
		respRegisterCh: make(chan *rkcy.RespChan, 10),
	}

	if respTarget != nil {
		var respConsumerCtx context.Context
		respConsumerCtx, akprod.respConsumerClose = context.WithCancel(akprod.ctx)
		wg.Add(1)
		go akprod.consumeResponseTopic(respConsumerCtx, wg)
	}

	return akprod
}

func (akprod *ApecsKafkaProducer) Close() {
	if akprod.respTarget != nil {
		akprod.respConsumerClose()
	}

	akprod.producersMtx.Lock()
	defer akprod.producersMtx.Unlock()
	for _, concernProds := range akprod.producers {
		for _, pdc := range concernProds {
			pdc.Close()
		}
	}

	akprod.producers = make(map[string]map[rkcy.StandardTopicName]rkcy.Producer)
}

func (akprod *ApecsKafkaProducer) GetProducer(
	concernName string,
	topicName rkcy.StandardTopicName,
	wg *sync.WaitGroup,
) (rkcy.Producer, error) {
	akprod.producersMtx.Lock()
	defer akprod.producersMtx.Unlock()

	concernProds, ok := akprod.producers[concernName]
	if !ok {
		concernProds = make(map[rkcy.StandardTopicName]rkcy.Producer)
		akprod.producers[concernName] = concernProds
	}
	pdc, ok := concernProds[topicName]
	if !ok {
		pdc = akprod.plat.NewProducer(
			akprod.ctx,
			concernName,
			string(topicName),
			wg,
		)

		if pdc == nil {
			return nil, fmt.Errorf(
				"ApecsKafkaProducer.GetProducer Brokers=%s Platform=%s Concern=%s Topic=%s: Failed to create Producer",
				akprod.plat.AdminBrokers(),
				akprod.plat.Name(),
				concernName,
				topicName,
			)
		}
		concernProds[topicName] = pdc
	}
	return rkcy.Producer(pdc), nil
}

func respondThroughChannel(txnId string, respCh chan *rkcypb.ApecsTxn, txn *rkcypb.ApecsTxn) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Str("TxnId", txnId).
				Msgf("recover while sending to respCh '%s'", r)
		}
	}()
	respCh <- txn
}

func (akprod *ApecsKafkaProducer) consumeResponseTopic(
	ctx context.Context,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	reqMap := make(map[string]*rkcy.RespChan)

	groupName := fmt.Sprintf("rkcy_response_%s", akprod.respTarget.Topic)

	kafkaLogCh := make(chan kafka.LogEvent)
	go rkcy.PrintKafkaLogs(ctx, kafkaLogCh)

	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        akprod.respTarget.Brokers,
		"group.id":                 groupName,
		"enable.auto.commit":       false,
		"enable.auto.offset.store": false,

		"go.logs.channel.enable": true,
		"go.logs.channel":        kafkaLogCh,
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Str("BoostrapServers", akprod.respTarget.Brokers).
			Str("GroupId", groupName).
			Msg("Unable to kafka.NewConsumer")
	}
	shouldCommit := false
	defer func() {
		log.Warn().
			Str("Topic", akprod.respTarget.Topic).
			Msgf("CONSUMER Closing...")
		if shouldCommit {
			_, err = cons.Commit()
			shouldCommit = false
			if err != nil {
				log.Error().
					Err(err).
					Msgf("Unable to commit")
			}
		}
		cons.Close()
		log.Warn().
			Str("Topic", akprod.respTarget.Topic).
			Msgf("CONSUMER CLOSED")
	}()

	err = cons.Assign([]kafka.TopicPartition{
		{
			Topic:     &akprod.respTarget.Topic,
			Partition: akprod.respTarget.Partition,
			Offset:    kafka.OffsetStored,
		},
	})
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to Assign")
	}

	firstMessage := true
	cleanupTicker := time.NewTicker(10 * time.Second)
	commitTicker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			log.Warn().
				Msg("consumeResponseTopic exiting, ctx.Done()")
			return
		case <-cleanupTicker.C:
			for txnId, respCh := range reqMap {
				now := time.Now()
				if now.Sub(respCh.StartTime) >= time.Second*60 {
					log.Warn().
						Str("TxnId", txnId).
						Msgf("Deleting request channel info, this is not normal and this transaction may have been lost")
					respondThroughChannel(txnId, respCh.RespCh, nil)
					delete(reqMap, txnId)
				}
			}
		case <-commitTicker.C:
			if shouldCommit {
				_, err = cons.Commit()
				shouldCommit = false
				if err != nil {
					log.Fatal().
						Err(err).
						Msgf("Unable to commit")
				}
			}
		default:
			msg, err := cons.ReadMessage(100 * time.Millisecond)

			// Read all registration messages here so we are sure to catch them before handling message
			moreRegistrations := true
			for moreRegistrations {
				select {
				case rch := <-akprod.respRegisterCh:
					_, ok := reqMap[rch.TxnId]
					if ok {
						log.Error().
							Str("TxnId", rch.TxnId).
							Msg("TxnId already registered for responses, replacing with new value")
					}
					reqMap[rch.TxnId] = rch
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
				// If this is the first message read, commit the
				// offset to current to ensure we have an offset in
				// kafka, and if we blow up and start again we will
				// not default to latest.
				if firstMessage {
					firstMessage = false
					log.Debug().
						Str("Topic", akprod.respTarget.Topic).
						Int32("Partition", akprod.respTarget.Partition).
						Int64("Offset", int64(msg.TopicPartition.Offset)).
						Msg("Initial commit to current offset")

					comOffs, err := cons.CommitOffsets([]kafka.TopicPartition{
						{
							Topic:     &akprod.respTarget.Topic,
							Partition: akprod.respTarget.Partition,
							Offset:    msg.TopicPartition.Offset,
						},
					})
					if err != nil {
						log.Fatal().
							Err(err).
							Str("Topic", akprod.respTarget.Topic).
							Int32("Partition", akprod.respTarget.Partition).
							Int64("Offset", int64(msg.TopicPartition.Offset)).
							Msgf("Unable to commit initial offset")
					}
					log.Debug().
						Str("Topic", akprod.respTarget.Topic).
						Int32("Partition", akprod.respTarget.Partition).
						Int64("Offset", int64(msg.TopicPartition.Offset)).
						Msgf("Initial offset committed: %+v", comOffs)
				}

				directive := rkcy.GetDirective(msg)
				if directive == rkcypb.Directive_APECS_TXN {
					txnId := rkcy.GetTraceId(msg)
					respCh, ok := reqMap[txnId]
					if !ok {
						log.Error().
							Str("TxnId", txnId).
							Msg("TxnId not found in reqMap")
					} else {
						delete(reqMap, txnId)
						txn := rkcypb.ApecsTxn{}
						err := proto.Unmarshal(msg.Value, &txn)
						if err != nil {
							log.Error().
								Err(err).
								Str("TxnId", txnId).
								Msg("Failed to Unmarshal ApecsTxn")
						} else {
							respondThroughChannel(txnId, respCh.RespCh, &txn)
						}
					}
				} else {
					log.Warn().
						Int("Directive", int(directive)).
						Msg("Invalid directive on ApecsTxn topic")
				}

				_, err = cons.StoreOffsets([]kafka.TopicPartition{
					{
						Topic:     &akprod.respTarget.Topic,
						Partition: akprod.respTarget.Partition,
						Offset:    msg.TopicPartition.Offset + 1,
					},
				})
				if err != nil {
					log.Fatal().
						Err(err).
						Msgf("Unable to store offsets %s/%d/%d", akprod.respTarget.Topic, akprod.respTarget.Partition, msg.TopicPartition.Offset+1)
				}
				shouldCommit = true
			}
		}
	}
}
