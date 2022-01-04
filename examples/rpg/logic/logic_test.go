// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package logic

import (
	"context"
	"testing"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/stream/offline"
)

func TestCharacter(t *testing.T) {
	rkcy.PrepLogging()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := offline.NewOfflinePlatformFromJson(ctx, gTestPlatformDef)
	if err != nil {
		t.Fatalf("Failed to NewOfflinePlatformFromJson: %s", err.Error())
	}
}

// All the clusters below are distinct, to make sure
// we handle those scenarios which are harder to test
// with the online kafka platform
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
            "cluster": "edge_response",
            "partitionCount": 1
          }
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
            "cluster": "player_process",
            "partitionCount": 1
          }
        },
        {
          "name": "storage",
          "current": {
            "cluster": "player_storage",
            "partitionCount": 1
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
            "cluster": "character_process",
            "partitionCount": 1
          }
        },
        {
          "name": "storage",
          "current": {
            "cluster": "character_storage",
            "partitionCount": 1
          }
        }
      ]
    }
  ],

  "clusters": [
    {
      "name": "admin",
      "brokers": "offline_admin:0",
      "isAdmin": true
    },
    {
      "name": "edge_response",
      "brokers": "offline_edge_response:0"
    },
    {
      "name": "player_process",
      "brokers": "offline_player_process:0"
    },
    {
      "name": "player_storage",
      "brokers": "offline_player_storage:0"
    },
    {
      "name": "character_process",
      "brokers": "offline_character_process:0"
    },
    {
      "name": "character_storage",
      "brokers": "offline_character_storage:0"
    }
  ],
  "storageTargets": [
    {
      "name": "offline-0",
      "type": "ramdb",
      "isPrimary": true
    }
  ]
}`)
