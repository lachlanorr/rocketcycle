// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"time"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type Consumer interface {
	Assign(partitions []kafka.TopicPartition) error
	Close() error
	Commit() ([]kafka.TopicPartition, error)
	CommitOffsets(offsets []kafka.TopicPartition) ([]kafka.TopicPartition, error)
	QueryWatermarkOffsets(topic string, partition int32, timeoutMs int) (int64, int64, error)
	ReadMessage(timeout time.Duration) (*kafka.Message, error)
	StoreOffsets(offsets []kafka.TopicPartition) ([]kafka.TopicPartition, error)
}
