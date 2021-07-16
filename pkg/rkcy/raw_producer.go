// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
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

var gProducers = make(map[string]*ChanneledProducer)
var gProducersMtx = &sync.Mutex{}

func getProducerCh(brokers string) ProducerCh {
	gProducersMtx.Lock()
	defer gProducersMtx.Unlock()

	cp, ok := gProducers[brokers]
	if ok {
		return cp.Ch
	}

	var err error
	cp = &ChanneledProducer{Brokers: brokers}
	cp.Prod, err = kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
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
						gPlatformImpl.Telem.RecordProduceError(
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
	gProducers[brokers] = cp

	go runProducer(cp)
	return cp.Ch
}

func closeProducer(cp *ChanneledProducer) {
	log.Info().Msgf("Closing producer for %s", cp.Brokers)
	if cp.Ch != nil {
		close(cp.Ch)
	}
	if cp.Prod != nil {
		cp.Prod.Flush(5 * 1000)
		cp.Prod.Close()
	}
}

func runProducer(cp *ChanneledProducer) {
	defer closeProducer(cp)

	for {
		select {
		case msg := <-cp.Ch:
			err := cp.Prod.Produce(msg, nil)
			if err != nil {
				traceId := GetTraceId(msg)
				if traceId != "" {
					gPlatformImpl.Telem.RecordProduceError(
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
