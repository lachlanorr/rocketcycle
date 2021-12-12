// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type AdminClient interface {
	GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error)
	CreateTopics(
		ctx context.Context,
		topics []kafka.TopicSpecification,
		options ...kafka.CreateTopicsAdminOption,
	) ([]kafka.TopicResult, error)
	Close()
}
