// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package consumer

import (
	"time"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type KafkaConsumer struct {
	cons *kafka.Consumer
}

func NewKafkaConsumer(bootstrapServers string, groupName string, logCh chan kafka.LogEvent) (*KafkaConsumer, error) {
	cons, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":        bootstrapServers,
		"group.id":                 groupName,
		"enable.auto.commit":       false,
		"enable.auto.offset.store": false,

		"go.logs.channel.enable": true,
		"go.logs.channel":        logCh,
	})

	if err != nil {
		return nil, err
	}

	return &KafkaConsumer{
		cons: cons,
	}, nil
}

func (kcons *KafkaConsumer) Assign(partitions []kafka.TopicPartition) error {
	return kcons.cons.Assign(partitions)
}

func (kcons *KafkaConsumer) Close() {
	kcons.cons.Close()
}

func (kcons *KafkaConsumer) Commit() ([]kafka.TopicPartition, error) {
	return kcons.cons.Commit()
}

func (kcons *KafkaConsumer) CommitOffsets(offsets []kafka.TopicPartition) ([]kafka.TopicPartition, error) {
	return kcons.cons.CommitOffsets(offsets)
}

func (kcons *KafkaConsumer) QueryWatermarkOffsets(topic string, partition int32, timeoutMs int) (int64, int64, error) {
	return kcons.cons.QueryWatermarkOffsets(topic, partition, timeoutMs)
}

func (kcons *KafkaConsumer) ReadMessage(timeout time.Duration) (*kafka.Message, error) {
	return kcons.cons.ReadMessage(timeout)
}

func (kcons *KafkaConsumer) StoreOffsets(offsets []kafka.TopicPartition) ([]kafka.TopicPartition, error) {
	return kcons.cons.StoreOffsets(offsets)
}
