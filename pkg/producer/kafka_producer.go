// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package producer

import (
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

type KafkaProducer struct {
	prod *kafka.Producer
}

func NewKafkaProducer(bootstrapServers string, logCh chan kafka.LogEvent) (rkcy.Producer, error) {
	prod, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":  bootstrapServers,
		"acks":               -1,     // acks required from all in-sync replicas
		"message.timeout.ms": 600000, // 10 minutes

		"go.logs.channel.enable": true,
		"go.logs.channel":        logCh,
	})

	if err != nil {
		return nil, err
	}

	return &KafkaProducer{
		prod: prod,
	}, nil
}

func (kprod *KafkaProducer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	return kprod.prod.Produce(msg, deliveryChan)
}

func (kprod *KafkaProducer) Close() {
	kprod.prod.Close()
}

func (kprod *KafkaProducer) Events() chan kafka.Event {
	return kprod.prod.Events()
}

func (kprod *KafkaProducer) Flush(timeoutMs int) int {
	return kprod.prod.Flush(timeoutMs)
}
