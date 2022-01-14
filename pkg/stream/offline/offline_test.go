// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package offline

import (
	"fmt"
	"testing"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"
)

func TestTopic(t *testing.T) {
	topic := NewTopic("foo", 3)
	if len(topic.partitions) != 3 {
		t.Fatalf("Invalid partition count, expecting 3 vs %d", len(topic.partitions))
	}

	for pi, part := range topic.partitions {
		if part.Len() != 0 {
			t.Fatalf("Empty partition with non-zero Len(): %d", part.Len())
		}

		for mi := 0; mi < 10; mi++ {
			part.Produce(
				&kafka.Message{
					Value: []byte(fmt.Sprintf("Message %d.%d", pi, mi)),
				},
			)
		}
	}

	for pi, part := range topic.partitions {
		if part.Len() != 10 {
			t.Fatalf("Partition with wrong Len(): %d.%d", pi, part.Len())
		}

		msg := part.GetMessage(-1)
		if msg != nil {
			t.Fatalf("Non-nil GetMessage() for -1 offset")
		}

		msg = part.GetMessage(10)
		if msg != nil {
			t.Fatalf("Non-nil GetMessage() for out of bounds offset")
		}

		for mi := 0; mi < 10; mi++ {
			msg = part.GetMessage(kafka.Offset(mi))
			if msg.Timestamp.IsZero() {
				t.Fatalf("Timestamp not set in message %d.%d", pi, mi)
			}
			if msg.TimestampType != kafka.TimestampLogAppendTime {
				t.Fatalf("Bad TimestampType in message %d.%d", pi, mi)
			}

			expectedVal := fmt.Sprintf("Message %d.%d", pi, mi)
			if string(msg.Value) != expectedVal {
				t.Fatalf("Unexpected Value for message %d.%d, expecting '%s' vs '%s'", pi, mi, expectedVal, string(msg.Value))
			}
		}
	}

	if len(topic.partitions) != 3 {
		t.Fatalf("Invalid partition count, expecting 3 vs %d", len(topic.partitions))
	}
}
