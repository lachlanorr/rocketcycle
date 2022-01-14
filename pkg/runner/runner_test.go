// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package runner

import (
	"testing"
)

func TestOfflineRunner(t *testing.T) {
	ornr, err := NewOfflineRunner(gTestPlatformDef, gTestConfig, nil)
	if err != nil {
		t.Fatalf("NewOfflineRunner error: %s", err.Error())
	}

	// validate all topics are created
	for _, topDet := range gTopicDetails {
		clus, err := ornr.OfflineManager().GetCluster(topDet.brokers)
		if err != nil {
			t.Fatalf("GetCluster error: %s", err.Error())
		}

		topic, err := clus.GetTopic(topDet.name)
		if err != nil {
			t.Fatalf("Topic not found '%s'", err.Error())
		}

		if topic.PartitionCount() != topDet.count {
			t.Fatalf("Bad partition count %s, %d vs %d", topDet.name, topic.PartitionCount(), topDet.count)
		}

		for i := int32(0); i < topDet.count; i++ {
			part, err := clus.GetPartition(topDet.name, i)
			if err != nil {
				t.Fatalf("GetPartition error %s.%d: %s", topDet.name, i, err.Error())
			}
			if part.Topic().Name() != topDet.name {
				t.Errorf("Bad topic name %s.%d, '%s' vs '%s'", topDet.name, i, part.Topic().Name(), topDet.name)
			}
			if part.Index() != i {
				t.Errorf("Bad partition index %s.%d, %d vs %d", topDet.name, i, part.Index(), i)
			}

			if part.Topic().Name() == "rkcy.rpg.test.platform" {
				if part.Len() != 1 {
					t.Errorf("Non-one message count %s.%d, %d", topDet.name, i, part.Len())
				}
			} else if part.Len() != 0 {
				t.Errorf("Non-zero message count %s.%d, %d", topDet.name, i, part.Len())
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

var gTestConfig = []byte(`{
  "testString": "testStringVal",
  "testBool": true,
  "testInt": 1234,
  "testDouble": 1.234,
  "limitsList": [
    {
      "id": "48772bc4-81bf-4273-9e4e-5ac56473ccc0",
      "maxCharactersPerPlayer": 5
    }
  ]
}`)

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
