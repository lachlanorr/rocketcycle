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

type KafkaApecsProducer struct {
	ctx               context.Context
	plat              rkcy.Platform
	respTarget        *rkcypb.TopicTarget
	producers         map[string]map[rkcy.StandardTopicName]rkcy.ManagedProducer
	producersMtx      sync.Mutex
	respRegisterCh    chan *rkcy.RespChan
	respConsumerClose context.CancelFunc
}

func (kaprod *KafkaApecsProducer) Platform() rkcy.Platform {
	return kaprod.plat
}

func (kaprod *KafkaApecsProducer) ResponseTarget() *rkcypb.TopicTarget {
	return kaprod.respTarget
}

func (kaprod *KafkaApecsProducer) RegisterResponseChannel(respCh *rkcy.RespChan) {
	kaprod.respRegisterCh <- respCh
}

func NewKafkaApecsProducer(
	ctx context.Context,
	plat rkcy.Platform,
	respTarget *rkcypb.TopicTarget,
	wg *sync.WaitGroup,
) *KafkaApecsProducer {
	kaprod := &KafkaApecsProducer{
		ctx:        ctx,
		plat:       plat,
		respTarget: respTarget,

		producers:      make(map[string]map[rkcy.StandardTopicName]rkcy.ManagedProducer),
		respRegisterCh: make(chan *rkcy.RespChan, 10),
	}

	if respTarget != nil {
		var respConsumerCtx context.Context
		respConsumerCtx, kaprod.respConsumerClose = context.WithCancel(kaprod.ctx)
		wg.Add(1)
		go consumeResponseTopic(respConsumerCtx, plat, kaprod.respTarget, kaprod.respRegisterCh, wg)
	}

	return kaprod
}

func (kaprod *KafkaApecsProducer) Close() {
	if kaprod.respTarget != nil {
		kaprod.respConsumerClose()
	}

	kaprod.producersMtx.Lock()
	defer kaprod.producersMtx.Unlock()
	for _, concernProds := range kaprod.producers {
		for _, pdc := range concernProds {
			pdc.Close()
		}
	}

	kaprod.producers = make(map[string]map[rkcy.StandardTopicName]rkcy.ManagedProducer)
}

func (kaprod *KafkaApecsProducer) GetManagedProducer(
	concernName string,
	topicName rkcy.StandardTopicName,
	wg *sync.WaitGroup,
) (rkcy.ManagedProducer, error) {
	kaprod.producersMtx.Lock()
	defer kaprod.producersMtx.Unlock()

	concernProds, ok := kaprod.producers[concernName]
	if !ok {
		concernProds = make(map[rkcy.StandardTopicName]rkcy.ManagedProducer)
		kaprod.producers[concernName] = concernProds
	}
	mprod, ok := concernProds[topicName]
	if !ok {
		mprod = kaprod.plat.NewManagedProducer(
			kaprod.ctx,
			concernName,
			string(topicName),
			wg,
		)

		if mprod == nil {
			return nil, fmt.Errorf(
				"KafkaApecsProducer.GetManagedProducer Brokers=%s Platform=%s Concern=%s Topic=%s: Failed to create Producer",
				kaprod.plat.AdminBrokers(),
				kaprod.plat.Name(),
				concernName,
				topicName,
			)
		}
		concernProds[topicName] = mprod
	}
	return mprod, nil
}

func respondThroughChannel(txnId string, respCh chan<- *rkcypb.ApecsTxn, txn *rkcypb.ApecsTxn) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Str("TxnId", txnId).
				Msgf("recover while sending to respCh '%s'", r)
		}
	}()
	respCh <- txn
}

func consumeResponseTopic(
	ctx context.Context,
	plat rkcy.Platform,
	respTarget *rkcypb.TopicTarget,
	respRegisterCh <-chan *rkcy.RespChan,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	reqMap := make(map[string]*rkcy.RespChan)

	groupName := fmt.Sprintf("rkcy_response_%s", respTarget.Topic)

	kafkaLogCh := make(chan kafka.LogEvent)
	go rkcy.PrintKafkaLogs(ctx, kafkaLogCh)

	cons, err := plat.NewConsumer(respTarget.Brokers, groupName, kafkaLogCh)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("BootstrapServers", respTarget.Brokers).
			Str("GroupId", groupName).
			Msg("Unable to plat.NewConsumer")
	}
	shouldCommit := false
	defer func() {
		log.Warn().
			Str("Topic", respTarget.Topic).
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
		err := cons.Close()
		if err != nil {
			log.Error().
				Err(err).
				Str("Topic", respTarget.Topic).
				Msgf("Error during consumer.Close()")
		}
		log.Warn().
			Str("Topic", respTarget.Topic).
			Msgf("CONSUMER CLOSED")
	}()

	err = cons.Assign([]kafka.TopicPartition{
		{
			Topic:     &respTarget.Topic,
			Partition: respTarget.Partition,
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
				case rch := <-respRegisterCh:
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
						Str("Topic", respTarget.Topic).
						Int32("Partition", respTarget.Partition).
						Int64("Offset", int64(msg.TopicPartition.Offset)).
						Msg("Initial commit to current offset")

					comOffs, err := cons.CommitOffsets([]kafka.TopicPartition{
						{
							Topic:     &respTarget.Topic,
							Partition: respTarget.Partition,
							Offset:    msg.TopicPartition.Offset,
						},
					})
					if err != nil {
						log.Fatal().
							Err(err).
							Str("Topic", respTarget.Topic).
							Int32("Partition", respTarget.Partition).
							Int64("Offset", int64(msg.TopicPartition.Offset)).
							Msgf("Unable to commit initial offset")
					}
					log.Debug().
						Str("Topic", respTarget.Topic).
						Int32("Partition", respTarget.Partition).
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
						Topic:     &respTarget.Topic,
						Partition: respTarget.Partition,
						Offset:    msg.TopicPartition.Offset + 1,
					},
				})
				if err != nil {
					log.Fatal().
						Err(err).
						Msgf("Unable to store offsets %s/%d/%d", respTarget.Topic, respTarget.Partition, msg.TopicPartition.Offset+1)
				}
				shouldCommit = true
			}
		}
	}
}
