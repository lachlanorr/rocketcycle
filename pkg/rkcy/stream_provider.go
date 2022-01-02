// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"time"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type ProducerCh chan *kafka.Message

type StreamProvider interface {
	NewConsumer(brokers string, groupName string, logCh chan kafka.LogEvent) (Consumer, error)
	NewProducer(brokers string, logCh chan kafka.LogEvent) (Producer, error)
	NewAdminClient(brokers string) (AdminClient, error)
}

type Consumer interface {
	Assign(partitions []kafka.TopicPartition) error
	Close() error
	Commit() ([]kafka.TopicPartition, error)
	CommitOffsets(offsets []kafka.TopicPartition) ([]kafka.TopicPartition, error)
	QueryWatermarkOffsets(topic string, partition int32, timeoutMs int) (int64, int64, error)
	ReadMessage(timeout time.Duration) (*kafka.Message, error)
	StoreOffsets(offsets []kafka.TopicPartition) ([]kafka.TopicPartition, error)
}

type Producer interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	Close()
	Events() chan kafka.Event
	Flush(timeoutMs int) int
}

type AdminClient interface {
	GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error)
	CreateTopics(
		ctx context.Context,
		topics []kafka.TopicSpecification,
		options ...kafka.CreateTopicsAdminOption,
	) ([]kafka.TopicResult, error)
	Close()
}
