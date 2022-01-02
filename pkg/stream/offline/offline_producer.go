// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package offline

import (
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type OfflineProducer struct {
	cluster *Cluster
	events  chan kafka.Event
}

func NewOfflineProducer(cluster *Cluster) *OfflineProducer {
	oprod := &OfflineProducer{
		cluster: cluster,
		events:  make(chan kafka.Event, 100),
	}
	return oprod
}

func (oprod *OfflineProducer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	part, err := oprod.cluster.GetPartition(*msg.TopicPartition.Topic, msg.TopicPartition.Partition)
	if err != nil {
		return err
	}
	part.Produce(msg)
	return nil
}

func (*OfflineProducer) Close() {
	// no-op
}

func (oprod *OfflineProducer) Events() chan kafka.Event {
	return oprod.events
}

func (*OfflineProducer) Flush(timeoutMs int) int {
	// no-op
	return 0
}
