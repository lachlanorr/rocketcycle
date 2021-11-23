// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type ProducerCh chan *kafka.Message

type ChanneledProducer struct {
	Brokers string
	Prod    *kafka.Producer
	Ch      ProducerCh
}

type RawProducer struct {
	producers    map[string]*ChanneledProducer
	producersMtx *sync.Mutex
	telem        *Telemetry
}

func NewRawProducer(telem *Telemetry) *RawProducer {
	return &RawProducer{
		producers:    make(map[string]*ChanneledProducer),
		producersMtx: &sync.Mutex{},
		telem:        telem,
	}
}

func (rawProd *RawProducer) getProducerCh(ctx context.Context, brokers string, wg *sync.WaitGroup) ProducerCh {
	rawProd.producersMtx.Lock()
	defer rawProd.producersMtx.Unlock()

	cp, ok := rawProd.producers[brokers]
	if ok {
		return cp.Ch
	}

	var err error
	cp = &ChanneledProducer{Brokers: brokers}
	kafkaLogCh := make(chan kafka.LogEvent)
	go printKafkaLogs(ctx, kafkaLogCh)

	cp.Prod, err = kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":  brokers,
		"acks":               -1,     // acks required from all in-sync replicas
		"message.timeout.ms": 600000, // 10 minutes

		"go.logs.channel.enable": true,
		"go.logs.channel":        kafkaLogCh,
	})

	if err != nil {
		log.Fatal().
			Err(err).
			Msgf("Failed to create producer to %s", brokers)
		return nil
	}
	go func() {
		for e := range cp.Prod.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					traceId := GetTraceId(ev)
					if traceId != "" {
						rawProd.telem.RecordProduceError(
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

	cp.Ch = make(ProducerCh)
	rawProd.producers[brokers] = cp

	wg.Add(1)
	go runProducer(ctx, cp, rawProd.telem, wg)
	return cp.Ch
}

func closeProducer(cp *ChanneledProducer) {
	log.Warn().
		Str("Brokers", cp.Brokers).
		Msg("PRODUCER Closing...")
	if cp.Ch != nil {
		close(cp.Ch)
	}
	if cp.Prod != nil {
		cp.Prod.Flush(60 * 1000)
		cp.Prod.Close()
	}
	log.Warn().
		Str("Brokers", cp.Brokers).
		Msg("PRODUCER CLOSED")
}

func runProducer(ctx context.Context, cp *ChanneledProducer, telem *Telemetry, wg *sync.WaitGroup) {
	defer wg.Done()
	defer closeProducer(cp)

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-cp.Ch:
			err := cp.Prod.Produce(msg, nil)
			if err != nil {
				traceId := GetTraceId(msg)
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
					Str("Brokers", cp.Brokers).
					Msgf("Produce failed: %+v", msg)
			}
		}
	}
}
