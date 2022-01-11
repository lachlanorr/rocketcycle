// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package offline

import (
	"context"
	"fmt"
	"testing"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
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

func TestPlatform(t *testing.T) {
	rkcy.PrepLogging()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	plat, err := NewOfflinePlatformFromJson(ctx, "TestPlatform", gTestPlatformDef, nil)
	if err != nil {
		t.Fatalf("NewOfflinePlatformFromJson error: %s", err.Error())
	}

	ostrmprov, ok := plat.StreamProvider().(*OfflineStreamProvider)
	if !ok {
		t.Fatalf("OfflinePlatform without OfflineStreamProvider")
	}

	// validate all topics are created
	for _, topDet := range gTopicDetails {
		clus, err := ostrmprov.mgr.GetCluster(topDet.brokers)
		if err != nil {
			t.Fatalf("GetCluster error: %s", err.Error())
		}

		topic, ok := clus.topics[topDet.name]
		if !ok {
			t.Fatalf("Topic not found '%s'", topDet.name)
		}

		if len(topic.partitions) != int(topDet.count) {
			t.Fatalf("Bad partition count %s, %d vs %d", topDet.name, len(topic.partitions), topDet.count)
		}

		for i := int32(0); i < topDet.count; i++ {
			part, err := clus.GetPartition(topDet.name, i)
			if err != nil {
				t.Fatalf("GetPartition error %s.%d: %s", topDet.name, i, err.Error())
			}
			if part.topic.name != topDet.name {
				t.Errorf("Bad topic name %s.%d, '%s' vs '%s'", topDet.name, i, part.topic.name, topDet.name)
			}
			if part.index != i {
				t.Errorf("Bad partition index %s.%d, %d vs %d", topDet.name, i, part.index, i)
			}

			if part.topic.name == "rkcy.rpg.test.platform" {
				if len(part.messages) != 1 {
					t.Errorf("Non-one message count %s.%d, %d", topDet.name, i, len(part.messages))
				}
			} else if len(part.messages) != 0 {
				t.Errorf("Non-zero message count %s.%d, %d", topDet.name, i, len(part.messages))
			}
		}
	}
}

type TopicDetails struct {
	brokers string
	name    string
	count   int32
}

var gTopicDetails = []TopicDetails{
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.platform",
		count:   1,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.config",
		count:   1,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.producers",
		count:   1,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.consumers",
		count:   1,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.edge.GENERAL.response.0001",
		count:   20,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.edge.GENERAL.admin.0001",
		count:   1,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.edge.GENERAL.error.0001",
		count:   1,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.Player.APECS.process.0001",
		count:   1,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.Player.APECS.storage.0001",
		count:   3,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.Player.APECS.admin.0001",
		count:   1,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.Player.APECS.error.0001",
		count:   1,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.Player.APECS.complete.0001",
		count:   1,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.Player.APECS.storage-scnd.0001",
		count:   1,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.Character.APECS.process.0001",
		count:   5,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.Character.APECS.storage.0001",
		count:   10,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.Character.APECS.admin.0001",
		count:   1,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.Character.APECS.error.0001",
		count:   1,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.Character.APECS.complete.0001",
		count:   1,
	},
	{
		brokers: "offline:9092",
		name:    "rkcy.rpg.test.Character.APECS.storage-scnd.0001",
		count:   1,
	},
}

var gTestPlatformDef = []byte(`{
  "name": "rpg",
  "environment": "test",
  "concerns": [
    {
      "type": "GENERAL",
      "name": "edge",

      "topics": [
        {
          "name": "response",
          "current": {
            "partitionCount": 20
          },
          "consumerPrograms": [
            {
              "name": "./@platform",
              "args": ["edge", "serve",
                       "-e", "@environment",
                       "-t", "@topic",
                       "-p", "@partition",
                       "--admin_brokers", "@admin_brokers",
                       "--consumer_brokers", "@consumer_brokers",
                       "--http_addr", ":1135@partition",
                       "--grpc_addr", ":1136@partition",
                       "--stream", "@stream",
                       "--otelcol_endpoint", "@otelcol_endpoint"
                      ],
              "abbrev": "edge/@partition",
              "tags": {"service.name": "rkcy.@platform.@environment.@concern"}
            }
          ]
        }
      ]
    },
    {
      "type": "APECS",
      "name": "Player",

      "topics": [
        {
          "name": "process",
          "current": {
            "partitionCount": 1
          }
        },
        {
          "name": "storage",
          "current": {
            "partitionCount": 3
          }
        }
      ]
    },
    {
      "type": "APECS",
      "name": "Character",

      "topics": [
        {
          "name": "process",
          "current": {
            "partitionCount": 5
          }
        },
        {
          "name": "storage",
          "current": {
            "partitionCount": 10
          }
        }
      ]
    }
  ],

  "clusters": [
    {
      "name": "offline",
      "brokers": "offline:9092",
      "isAdmin": true
    }
  ],

  "storageTargets": [
    {
      "name": "offline",
      "type": "ramdb",
      "isPrimary": true
    }
  ]
}`)
