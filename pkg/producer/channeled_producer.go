// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package producer

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

type ChanneledProducer struct {
	plat    rkcy.Platform
	brokers string
	ch      rkcy.ProducerCh
	prod    rkcy.Producer
}

func NewChanneledProducer(
	ctx context.Context,
	plat rkcy.Platform,
	brokers string,
) (*ChanneledProducer, error) {
	cp := &ChanneledProducer{
		plat:    plat,
		brokers: brokers,
		ch:      make(rkcy.ProducerCh),
	}

	kafkaLogCh := make(chan kafka.LogEvent)
	go rkcy.PrintKafkaLogs(ctx, kafkaLogCh)

	var err error
	cp.prod, err = plat.NewProducer(brokers, kafkaLogCh)

	if err != nil {
		return nil, err
	}
	go func() {
		for e := range cp.prod.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					traceId := rkcy.GetTraceId(ev)
					if traceId != "" {
						cp.plat.Telem().RecordProduceError(
							"Delivery",
							traceId,
							*ev.TopicPartition.Topic,
							ev.TopicPartition.Partition,
							ev.TopicPartition.Error,
						)
					}
					log.Error().
						Err(ev.TopicPartition.Error).
						Str("Brokers", brokers).
						Msgf("Delivery failed: %+v", ev)
				}
			}
		}
	}()

	return cp, nil
}

func (cp *ChanneledProducer) Close() {
	log.Warn().
		Str("Brokers", cp.brokers).
		Msg("PRODUCER Closing...")
	if cp.ch != nil {
		close(cp.ch)
	}
	if cp.prod != nil {
		cp.prod.Flush(60 * 1000)
		cp.prod.Close()
	}
	log.Warn().
		Str("Brokers", cp.brokers).
		Msg("PRODUCER CLOSED")
}

func (cp *ChanneledProducer) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer cp.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-cp.ch:
			err := cp.prod.Produce(msg, nil)
			if err != nil {
				traceId := rkcy.GetTraceId(msg)
				if traceId != "" {
					cp.plat.Telem().RecordProduceError(
						"Produce",
						traceId,
						*msg.TopicPartition.Topic,
						msg.TopicPartition.Partition,
						msg.TopicPartition.Error,
					)
				}
				log.Error().
					Err(err).
					Str("Brokers", cp.brokers).
					Msgf("Produce failed: %+v", msg)
			}
		}
	}
}
