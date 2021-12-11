// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"sync"
	"time"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type RespChan struct {
	TxnId     string
	RespCh    chan *rkcypb.ApecsTxn
	StartTime time.Time
}

type Producer interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	Close()
	Events() chan kafka.Event
	Flush(timeoutMs int) int
}

type ManagedProducer interface {
	Produce(
		directive rkcypb.Directive,
		traceParent string,
		key []byte,
		value []byte,
		deliveryChan chan kafka.Event,
	)
	Close()
}

type ApecsProducer interface {
	Platform() Platform
	ResponseTarget() *rkcypb.TopicTarget
	RegisterResponseChannel(respCh *RespChan)

	GetManagedProducer(
		concernName string,
		topicName StandardTopicName,
		wg *sync.WaitGroup,
	) (ManagedProducer, error)
}
