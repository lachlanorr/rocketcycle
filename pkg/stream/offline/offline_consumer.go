// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package offline

import (
	"fmt"
	"sync"
	"time"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

type PartitionOffset struct {
	part   *Partition
	offset kafka.Offset
}

type OfflineConsumer struct {
	clus      *Cluster
	readCount int
	partOffs  []*PartitionOffset
	mtx       sync.Mutex
}

func NewOfflineConsumer(clus *Cluster) *OfflineConsumer {
	ocons := &OfflineConsumer{
		clus:      clus,
		readCount: 0,
	}
	return ocons
}

func (ocons *OfflineConsumer) Assign(partitions []kafka.TopicPartition) error {
	ocons.mtx.Lock()
	defer ocons.mtx.Unlock()

	ocons.partOffs = make([]*PartitionOffset, len(partitions))
	for i, tp := range partitions {
		part, err := ocons.clus.GetPartition(*tp.Topic, tp.Partition)
		if err != nil {
			return err
		}
		ocons.partOffs[i] = &PartitionOffset{
			part:   part,
			offset: tp.Offset,
		}
	}
	return nil
}

func (ocons *OfflineConsumer) Close() error {
	// no-op
	return nil
}

func (ocons *OfflineConsumer) Commit() ([]kafka.TopicPartition, error) {
	ocons.mtx.Lock()
	defer ocons.mtx.Unlock()

	// we commit instantly, no need to periodically commit back to
	// brokers, just return our offsets
	tps := make([]kafka.TopicPartition, len(ocons.partOffs))
	for i, partOff := range ocons.partOffs {
		tps[i].Topic = &partOff.part.topic.name
		tps[i].Partition = partOff.part.index
		tps[i].Offset = partOff.offset
	}
	return tps, nil
}

func (ocons *OfflineConsumer) CommitOffsets(offsets []kafka.TopicPartition) ([]kafka.TopicPartition, error) {
	// We have no distinction between local offset store and brokers offset store
	return ocons.StoreOffsets(offsets)
}

func (ocons *OfflineConsumer) QueryWatermarkOffsets(topic string, partition int32, timeoutMs int) (int64, int64, error) {
	part, err := ocons.clus.GetPartition(topic, partition)
	if err != nil {
		return 0, 0, err
	}
	return 0, part.Len(), nil
}

func (ocons *OfflineConsumer) ReadMessage(timeout time.Duration) (*kafka.Message, error) {
	ocons.mtx.Lock()
	defer ocons.mtx.Unlock()

	if ocons.partOffs == nil {
		return nil, fmt.Errorf("No assignment")
	}

	start := time.Now()
	for {
		if timeout != 0 && time.Now().Sub(start) >= timeout {
			break
		}

		// cycle through assignments
		ocons.readCount++
		partOff := ocons.partOffs[ocons.readCount%len(ocons.partOffs)]
		kMsg := partOff.part.GetMessage(partOff.offset)
		if kMsg != nil {
			partOff.offset++
			return kMsg, nil
		} else {
			time.Sleep(time.Millisecond * 100)
		}
	}

	// return same error type as kafka.Consumer.ReadMessage on timeout
	return nil, kafka.NewError(kafka.ErrTimedOut, "kafka.ErrTimedOut", false)
}

func (ocons *OfflineConsumer) StoreOffsets(offsets []kafka.TopicPartition) ([]kafka.TopicPartition, error) {
	ocons.mtx.Lock()
	defer ocons.mtx.Unlock()

	partOffs := make([]*PartitionOffset, len(offsets))
	for i, tp := range offsets {
		for _, partOff := range ocons.partOffs {
			if partOff.part.topic.name == *tp.Topic && partOff.part.index == tp.Partition {
				partOffs[i] = partOff
				break
			}
			if partOffs[i] == nil {
				return nil, fmt.Errorf("Invalid topic partition to commit, %s.%d", *tp.Topic, tp.Partition)
			}
		}
	}
	for i, tp := range offsets {
		partOffs[i].offset = tp.Offset
	}
	return offsets, nil
}
