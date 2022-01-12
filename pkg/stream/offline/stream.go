// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package offline

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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

func (part *Partition) Topic() *Topic {
	return part.topic
}

func (part *Partition) Index() int32 {
	return part.index
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

func (topic *Topic) Name() string {
	return topic.name
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
	name      string
	brokers   string
	brokersMd []kafka.BrokerMetadata
	topics    map[string]*Topic
	mtx       sync.Mutex
}

func NewCluster(name string, brokers string) (*Cluster, error) {
	clus := &Cluster{
		name:    name,
		brokers: brokers,
		topics:  make(map[string]*Topic),
	}

	brokerList := strings.Split(brokers, ",")
	clus.brokersMd = make([]kafka.BrokerMetadata, len(brokerList))
	for i, hostport := range brokerList {
		var (
			host string
			port int
			err  error
		)
		colpos := strings.Index(hostport, ":")
		if colpos != -1 {
			host = hostport[:colpos]
			port, err = strconv.Atoi(hostport[colpos+1:])
			if err != nil {
				return nil, err
			}
		} else {
			host = hostport
			port = 9092
		}

		clus.brokersMd[i] = kafka.BrokerMetadata{
			ID:   int32(i),
			Host: host,
			Port: port,
		}
	}

	return clus, nil
}

func (clus *Cluster) GetTopic(topicName string) (*Topic, error) {
	clus.mtx.Lock()
	defer clus.mtx.Unlock()

	topic, ok := clus.topics[topicName]
	if !ok {
		return nil, fmt.Errorf("Topic not found: %s", topicName)
	}
	return topic, nil
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
		res[i].Topic = topic.Topic

		if _, ok := clus.topics[topic.Topic]; ok {
			res[i].Error = kafka.NewError(kafka.ErrTopicAlreadyExists, "kafka.ErrTopicAlreadyExists", false)
			return res, nil
		}

		clus.topics[topic.Topic] = NewTopic(topic.Topic, topic.NumPartitions)
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

func NewManager(rtPlatDef *rkcy.RtPlatformDef) (*Manager, error) {
	mgr := &Manager{
		clusters: make(map[string]*Cluster),
	}

	mgr.adminBrokers = rtPlatDef.AdminCluster.Brokers

	var err error
	for _, clus := range rtPlatDef.Clusters {
		mgr.clusters[clus.Brokers], err = NewCluster(clus.Name, clus.Brokers)
		if err != nil {
			return nil, err
		}
	}

	return mgr, nil
}

func NewManagerFromJson(platformDefJson []byte) (*Manager, error) {
	rtPlatDef, err := rkcy.NewRtPlatformDefFromJson(platformDefJson)
	if err != nil {
		return nil, err
	}

	return NewManager(rtPlatDef)
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
