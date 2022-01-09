// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package stream

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/telem"
)

type ChanneledProducer struct {
	brokers string
	ch      rkcy.ProducerCh
	prod    rkcy.Producer
}

func NewChanneledProducer(
	ctx context.Context,
	strmprov rkcy.StreamProvider,
	brokers string,
) (*ChanneledProducer, error) {
	cp := &ChanneledProducer{
		brokers: brokers,
		ch:      make(rkcy.ProducerCh),
	}

	kafkaLogCh := make(chan kafka.LogEvent)
	go rkcy.PrintKafkaLogs(ctx, kafkaLogCh)

	var err error
	cp.prod, err = strmprov.NewProducer(brokers, kafkaLogCh)

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
						telem.RecordProduceError(
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

func (cp *ChanneledProducer) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		log.Trace().
			Str("Brokers", cp.brokers).
			Msg("PRODUCER Closing...")
		if cp.ch != nil {
			close(cp.ch)
		}
		if cp.prod != nil {
			cp.prod.Flush(60 * 1000)
			cp.prod.Close()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-cp.ch:
			err := cp.prod.Produce(msg, nil)
			if err != nil {
				traceId := rkcy.GetTraceId(msg)
				if traceId != "" {
					telem.RecordProduceError(
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
