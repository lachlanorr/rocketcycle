// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package stream

import (
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

type KafkaStreamProvider struct{}

func NewKafkaStreamProvider() *KafkaStreamProvider {
	return &KafkaStreamProvider{}
}

func (*KafkaStreamProvider) NewConsumer(brokers string, groupName string, logCh chan kafka.LogEvent) (rkcy.Consumer, error) {
	return kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        brokers,
		"group.id":                 groupName,
		"enable.auto.commit":       false,
		"enable.auto.offset.store": false,

		"go.logs.channel.enable": true,
		"go.logs.channel":        logCh,
	})
}

func (kstrmprov *KafkaStreamProvider) NewProducer(brokers string, logCh chan kafka.LogEvent) (rkcy.Producer, error) {
	return kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers":  brokers,
		"acks":               -1,     // acks required from all in-sync replicas
		"message.timeout.ms": 600000, // 10 minutes

		"go.logs.channel.enable": true,
		"go.logs.channel":        logCh,
	})
}

func (*KafkaStreamProvider) NewAdminClient(brokers string) (rkcy.AdminClient, error) {
	return kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": brokers,
	})
}
