// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package logic

import (
	"testing"

	"github.com/google/uuid"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/runner"
	"github.com/lachlanorr/rocketcycle/pkg/storage/ramdb"

	"github.com/lachlanorr/rocketcycle/examples/rpg/logic/txn"
	"github.com/lachlanorr/rocketcycle/examples/rpg/pb"
)

func NewTestingClientCode() *rkcy.ClientCode {
	clientCode := rkcy.NewClientCode()
	clientCode.AddStorageInit("ramdb", rkcy.StorageInitNoop)

	clientCode.AddLogicHandler("Player", &Player{})
	clientCode.AddLogicHandler("Character", &Character{})

	db := ramdb.NewRamDb()

	clientCode.AddCrudHandler("Player", "ramdb", pb.NewPlayerCrudHandlerRamDb(db))
	clientCode.AddCrudHandler("Character", "ramdb", pb.NewCharacterCrudHandlerRamDb(db))

	return clientCode
}

var gOrnr *runner.OfflineRunner

func init() {
	var err error
	gOrnr, err = runner.NewOfflineRunner(
		gTestPlatformDef,
		gTestConfig,
		NewTestingClientCode(),
	)
	if err != nil {
		panic(err.Error())
	}
}

func TestPlayer(t *testing.T) {
	{
		// verify id is assigned if not present
		playerNoId := *gTestPlayers["player0"]
		playerNoId.Id = ""
		res, err := gOrnr.ExecuteTxnSync(txn.CreatePlayer(&playerNoId))
		if err != nil {
			t.Fatalf("Failed to ExecuteTxnSync: %s", err.Error())
		}

		if res.Type != "Player" {
			t.Fatalf("res.Type not Player: %s", res.Type)
		}
		retPlayer, ok := res.Instance.(*pb.Player)
		if !ok {
			t.Fatal("res.Instance not type pb.Player")
		}

		if retPlayer.Id == "" || retPlayer.Id == gTestPlayers["player0"].Id {
			t.Fatal("id not assigned to player")
		}

		_, ok = res.Related.(*pb.PlayerRelated)
		if !ok {
			t.Fatal("res.Related not type pb.PlayerRelated")
		}
	}
}

func TestCharacter(t *testing.T) {
	{
		_, err := gOrnr.ExecuteTxnSync(txn.CreatePlayer(gTestPlayers["player0"]))
		if err != nil {
			t.Fatalf("Failed to ExecuteTxnSync: %s", err.Error())
		}

		// verify id is assigned if not present
		charNoId := *gTestChars["char0.0"]
		charNoId.Id = ""
		res, err := gOrnr.ExecuteTxnSync(txn.CreateCharacter(&charNoId))
		if err != nil {
			t.Fatalf("Failed to ExecuteTxnSync: %s", err.Error())
		}

		if res.Type != "Character" {
			t.Fatalf("res.Type not Character: %s", res.Type)
		}
		retChar, ok := res.Instance.(*pb.Character)
		if !ok {
			t.Fatal("res.Instance not type pb.Character")
		}

		if retChar.Id == "" || retChar.Id == gTestChars["char0.0"].Id {
			t.Fatal("id not assigned to player")
		}

		_, ok = res.Related.(*pb.CharacterRelated)
		if !ok {
			t.Fatal("res.Related not type pb.CharacterRelated")
		}
	}
}

var gTestPlayers = map[string]*pb.Player{
	"player0": &pb.Player{
		Id:       uuid.NewString(),
		Username: "player0",
		Active:   true,
		LimitsId: "48772bc4-81bf-4273-9e4e-5ac56473ccc0",
	},
	"player1": &pb.Player{
		Id:       uuid.NewString(),
		Username: "player1",
		Active:   true,
		LimitsId: "48772bc4-81bf-4273-9e4e-5ac56473ccc0",
	},
	"player2": &pb.Player{
		Id:       uuid.NewString(),
		Username: "player2",
		Active:   false,
		LimitsId: "48772bc4-81bf-4273-9e4e-5ac56473ccc0",
	},
}

var gTestChars = map[string]*pb.Character{
	"char0.0": &pb.Character{
		Id:       uuid.NewString(),
		PlayerId: gTestPlayers["player0"].Id,
		Fullname: "char0.0 Name",
		Active:   true,
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
