// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package offline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

//------------------------------------------------------------------------------

type Partition struct {
	topic    *Topic
	index    int32
	messages []*kafka.Message
	mtx      sync.Mutex
}

func NewPartition(topic *Topic, partition int32) *Partition {
	part := &Partition{
		topic:    topic,
		index:    partition,
		messages: make([]*kafka.Message, 0, 1000),
	}
	return part
}

func (part *Partition) Produce(msg *kafka.Message) {
	part.mtx.Lock()
	defer part.mtx.Unlock()

	msg.Timestamp = time.Now()
	msg.TimestampType = kafka.TimestampLogAppendTime
	msg.TopicPartition.Offset = kafka.Offset(len(part.messages))

	part.messages = append(part.messages, msg)
}

func (part *Partition) Len() int64 {
	part.mtx.Lock()
	defer part.mtx.Unlock()

	return int64(len(part.messages))
}

func (part *Partition) GetMessage(offset kafka.Offset) *kafka.Message {
	if offset < 0 {
		return nil
	}

	part.mtx.Lock()
	defer part.mtx.Unlock()

	if offset < kafka.Offset(len(part.messages)) {
		return part.messages[offset]
	}
	return nil
}

func (part *Partition) GetMetadata() kafka.PartitionMetadata {
	part.mtx.Lock()
	defer part.mtx.Unlock()

	return kafka.PartitionMetadata{
		ID: part.index,
	}
}

//------------------------------------------------------------------------------

type Topic struct {
	name       string
	partitions []*Partition
	mtx        sync.Mutex
}

func NewTopic(name string, partitionCount int) *Topic {
	topic := &Topic{
		name: name,
	}
	topic.partitions = make([]*Partition, partitionCount)
	for i, _ := range topic.partitions {
		topic.partitions[i] = NewPartition(topic, int32(i))
	}
	return topic
}

func (topic *Topic) PartitionCount() int32 {
	topic.mtx.Lock()
	defer topic.mtx.Unlock()

	return int32(len(topic.partitions))
}

func (topic *Topic) GetMetadata() kafka.TopicMetadata {
	topic.mtx.Lock()
	defer topic.mtx.Unlock()

	tmd := kafka.TopicMetadata{
		Topic: topic.name,
	}
	tmd.Partitions = make([]kafka.PartitionMetadata, len(topic.partitions))
	for i, part := range topic.partitions {
		tmd.Partitions[i] = part.GetMetadata()
	}
	return tmd
}

//------------------------------------------------------------------------------

type Cluster struct {
	name    string
	brokers string
	topics  map[string]*Topic
	mtx     sync.Mutex
}

func NewCluster(name string, brokers string) *Cluster {
	clust := &Cluster{
		name:    name,
		brokers: brokers,
		topics:  make(map[string]*Topic),
	}
	return clust
}

func (clus *Cluster) GetPartition(topicName string, partition int32) (*Partition, error) {
	clus.mtx.Lock()
	defer clus.mtx.Unlock()

	topic, ok := clus.topics[topicName]
	if !ok {
		return nil, fmt.Errorf("Topic not found: %s", topicName)
	}
	if partition >= int32(len(topic.partitions)) {
		return nil, fmt.Errorf("Partition out of range: %s.%d", topicName, partition)
	}
	return topic.partitions[partition], nil
}

// AdminClient methods
func (clus *Cluster) GetMetadata(topic *string, allTopics bool, timeoutMs int) (*kafka.Metadata, error) {
	clus.mtx.Lock()
	defer clus.mtx.Unlock()

	md := &kafka.Metadata{}

	md.Brokers = []kafka.BrokerMetadata{
		{
			ID:   0,
			Host: "offline",
			Port: 0,
		},
	}

	md.Topics = make(map[string]kafka.TopicMetadata)
	for name, topic := range clus.topics {
		md.Topics[name] = topic.GetMetadata()
	}

	return md, nil
}

func (clus *Cluster) CreateTopics(
	ctx context.Context,
	topics []kafka.TopicSpecification,
	options ...kafka.CreateTopicsAdminOption,
) ([]kafka.TopicResult, error) {
	res := make([]kafka.TopicResult, len(topics))
	for i, topic := range topics {
		clus.topics[topic.Topic] = NewTopic(topic.Topic, topic.NumPartitions)
		res[i].Topic = topic.Topic
		res[i].Error = kafka.NewError(kafka.ErrNoError, "kafka.ErrNoError", false)
	}
	return res, nil
}

func (clus *Cluster) Close() {
	// no-op
}

//------------------------------------------------------------------------------

type Manager struct {
	adminBrokers string
	clusters     map[string]*Cluster
	mtx          sync.Mutex
}

func NewManager(rtPlatDef *rkcy.RtPlatformDef) *Manager {
	mgr := &Manager{
		clusters: make(map[string]*Cluster),
	}

	mgr.adminBrokers = rtPlatDef.Clusters[rtPlatDef.AdminCluster].Brokers

	for _, cd := range rtPlatDef.Clusters {
		mgr.clusters[cd.Brokers] = NewCluster(cd.Name, cd.Brokers)
	}
	return mgr
}

func (mgr *Manager) GetCluster(brokers string) (*Cluster, error) {
	mgr.mtx.Lock()
	defer mgr.mtx.Unlock()

	clus, ok := mgr.clusters[brokers]
	if !ok {
		return nil, fmt.Errorf("No cluster with brokers: %s", brokers)
	}
	return clus, nil
}
