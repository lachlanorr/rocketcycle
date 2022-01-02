// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package logic

import (
	"context"
	"sync"
	"testing"

	"github.com/lachlanorr/rocketcycle/pkg/offline"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

func TestCharacter(t *testing.T) {
	rkcy.PrepLogging()

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	bailoutCh := make(chan bool)

	rtPlatDef, err := rkcy.NewRtPlatformDefFromJson(gTestPlatformDef, "rpg", "test")
	if err != nil {
		t.Fatalf("Failed to NewRtPlatformDefFromJson: %s", err.Error())
	}

	_, err := offline.NewOfflinePlatform(ctx, rtPlatDef, bailoutCh, &wg)
	if err != nil {
		t.Fatalf("Failed to NewOfflinePlatform: %s", err.Error())
	}

	//	res, err := apecs.ExecuteTxnSync(
	//		ctx,
	//		oplat,

	cancel()
	wg.Wait()
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
      "brokers": "offline_edge_response:0",
    },
    {
      "name": "character_process",
      "brokers": "offline_player_process:0",
    },
    {
      "name": "character_process",
      "brokers": "offline_player_storage:0",
    },
    {
      "name": "character_process",
      "brokers": "offline_character_process:0",
    },
    {
      "name": "character_process",
      "brokers": "offline_character_storage:0",
    }
  ],
  "storageTargets": [
    {
      "name": "offline-0",
      "type": "ramdb",
      "isPrimary": true,
    }
  ]
}`)
