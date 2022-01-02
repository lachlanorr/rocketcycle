// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"bytes"
	"context"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/lachlanorr/rocketcycle/pkg/jsonutils"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/pkg/watch"

	_ "github.com/lachlanorr/rocketcycle/examples/rpg/logic"
	_ "github.com/lachlanorr/rocketcycle/examples/rpg/pb"
)

func TestMarshalTxn(t *testing.T) {
	// get a protobuf from our sample json so we have a good start
	txn := &rkcypb.ApecsTxn{}
	err := protojson.Unmarshal(testTxnJson, txn)
	if err != nil {
		t.Fatal(err)
	}

	// serialize to json
	pjOpts := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}
	jsonBytes, err := pjOpts.Marshal(txn)
	if err != nil {
		t.Fatal(err)
	}

	var omap *jsonutils.OrderedMap
	err = jsonutils.UnmarshalOrdered(jsonBytes, &omap)
	if err != nil {
		t.Fatal(err)
	}

	marTxnJson, err := jsonutils.MarshalOrderedIndent(omap, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(marTxnJson, testTxnJson) {
		t.Errorf("Marshalled bytes do not match expected:\n\t%s\n\t-- vs.\n\t%s", string(marTxnJson), string(testTxnJson))
	}
}

func TestDecodeTxnOpaques(t *testing.T) {
	ctx := context.Background()

	// get a protobuf from our sample json so we have a good start
	txn := &rkcypb.ApecsTxn{}
	err := protojson.Unmarshal(testTxnJson, txn)
	if err != nil {
		t.Fatal(err)
	}

	pjOpts := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}
	marTxnJson, err := watch.DecodeTxnOpaques(ctx, txn, rkcy.GlobalConcernHandlerRegistry(), &pjOpts)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(marTxnJson, testTxnJsonDec) {
		t.Errorf("Marshalled bytes do not match expected:\n\t%s\n\t-- vs.\n\t%s", string(marTxnJson), string(testTxnJson))
	}
}

var testTxnJson = []byte(`{
  "id": "251a272b59ceefeb5bff6df5252a5805",
  "assocTxns": [
    {
      "id": "7f79dae03a46446c91623829a7fd1406",
      "type": "SECONDARY_STORAGE"
    },
    {
      "id": "71ee52c7e81044449a2babcd7bb33f7f",
      "type": "GENERATED_STORAGE"
    },
    {
      "id": "71ee52c7e81044449a2babcd7bb33f7f",
      "type": "GENERATED_STORAGE"
    },
    {
      "id": "288725b97df44c2dbd793b76f52d172c",
      "type": "GENERATED_STORAGE"
    },
    {
      "id": "288725b97df44c2dbd793b76f52d172c",
      "type": "GENERATED_STORAGE"
    }
  ],
  "responseTarget": {
    "brokers": "localhost:9092",
    "topic": "rkcy.rpg.dev.edge.GENERAL.response.0001",
    "partition": 0
  },
  "currentStepIdx": 4,
  "direction": "FORWARD",
  "uponError": "REVERT",
  "forwardSteps": [
    {
      "system": "PROCESS",
      "concern": "Character",
      "command": "ValidateCreate",
      "key": "",
      "payload": "EiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUaE0Z1bGxuYW1lXzE2NTg3ODkzODMgASoMCJYTELgPGMgqIK4q",
      "effectiveTime": "2021-12-13T12:29:42.486718Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "123"
      },
      "instance": "",
      "result": {
        "code": "OK",
        "processedTime": "2021-12-13T12:29:42.535509Z",
        "logEvents": [],
        "key": "",
        "payload": "EiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUaE0Z1bGxuYW1lXzE2NTg3ODkzODMgASoMCJYTELgPGMgqIK4q",
        "instance": "",
        "cmpdOffset": null
      }
    },
    {
      "system": "STORAGE",
      "concern": "Character",
      "command": "Create",
      "key": "",
      "payload": "EiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUaE0Z1bGxuYW1lXzE2NTg3ODkzODMgASoMCJYTELgPGMgqIK4q",
      "effectiveTime": "2021-12-13T12:29:42.486718Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "123"
      },
      "instance": "",
      "result": {
        "code": "OK",
        "processedTime": "2021-12-13T12:29:42.615506Z",
        "logEvents": [],
        "key": "b00d6b2a-9c6d-449d-9356-65cadc64237d",
        "payload": "cmtjeXkAAAAKJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBIkYTQzMWI5OTgtYzExYy00ODVlLWE5NDYtMjc5MWQwM2VhYjZlGhNGdWxsbmFtZV8xNjU4Nzg5MzgzIAEqDAiWExC4DxjIKiCuKg==",
        "instance": "",
        "cmpdOffset": {
          "generation": 1,
          "partition": 0,
          "offset": "123"
        }
      }
    },
    {
      "system": "PROCESS",
      "concern": "Character",
      "command": "RefreshInstance",
      "key": "b00d6b2a-9c6d-449d-9356-65cadc64237d",
      "payload": "cmtjeXkAAAAKJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBIkYTQzMWI5OTgtYzExYy00ODVlLWE5NDYtMjc5MWQwM2VhYjZlGhNGdWxsbmFtZV8xNjU4Nzg5MzgzIAEqDAiWExC4DxjIKiCuKg==",
      "effectiveTime": "2021-12-13T12:29:42.486718Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "124"
      },
      "instance": "",
      "result": {
        "code": "OK",
        "processedTime": "2021-12-13T12:29:42.700188Z",
        "logEvents": [],
        "key": "",
        "payload": "cmtjeXkAAAAKJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBIkYTQzMWI5OTgtYzExYy00ODVlLWE5NDYtMjc5MWQwM2VhYjZlGhNGdWxsbmFtZV8xNjU4Nzg5MzgzIAEqDAiWExC4DxjIKiCuKg==",
        "instance": "",
        "cmpdOffset": null
      }
    },
    {
      "system": "PROCESS",
      "concern": "Player",
      "command": "RequestRelated",
      "key": "a431b998-c11c-485e-a946-2791d03eab6e",
      "payload": "CglDaGFyYWN0ZXISJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBoGUGxheWVyInlya2N5eQAAAAokYjAwZDZiMmEtOWM2ZC00NDlkLTkzNTYtNjVjYWRjNjQyMzdkEiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUaE0Z1bGxuYW1lXzE2NTg3ODkzODMgASoMCJYTELgPGMgqIK4q",
      "effectiveTime": null,
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "118"
      },
      "instance": "",
      "result": {
        "code": "OK",
        "processedTime": "2021-12-13T12:29:42.740913Z",
        "logEvents": [],
        "key": "",
        "payload": "CgZQbGF5ZXISBlBsYXllchq0AXJrY3lBAAAACiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUSD1VzZXJfMTE3NjY5Mjk0NxgBEnEKJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBIkYTQzMWI5OTgtYzExYy00ODVlLWE5NDYtMjc5MWQwM2VhYjZlGhNGdWxsbmFtZV8xNjU4Nzg5MzgzIAEqDAiWExC4DxjIKiCuKg==",
        "instance": "",
        "cmpdOffset": null
      }
    },
    {
      "system": "PROCESS",
      "concern": "Character",
      "command": "RefreshRelated",
      "key": "b00d6b2a-9c6d-449d-9356-65cadc64237d",
      "payload": "CgZQbGF5ZXISBlBsYXllchq0AXJrY3lBAAAACiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUSD1VzZXJfMTE3NjY5Mjk0NxgBEnEKJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBIkYTQzMWI5OTgtYzExYy00ODVlLWE5NDYtMjc5MWQwM2VhYjZlGhNGdWxsbmFtZV8xNjU4Nzg5MzgzIAEqDAiWExC4DxjIKiCuKg==",
      "effectiveTime": "2021-12-13T12:29:42.741088Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "125"
      },
      "instance": "",
      "result": {
        "code": "OK",
        "processedTime": "2021-12-13T12:29:42.781459Z",
        "logEvents": [],
        "key": "",
        "payload": "cmtjeXkAAAAKJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBIkYTQzMWI5OTgtYzExYy00ODVlLWE5NDYtMjc5MWQwM2VhYjZlGhNGdWxsbmFtZV8xNjU4Nzg5MzgzIAEqDAiWExC4DxjIKiCuKhI5CiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUSD1VzZXJfMTE3NjY5Mjk0NxgB",
        "instance": "",
        "cmpdOffset": null
      }
    }
  ],
  "reverseSteps": []
}`)

var testTxnJsonDec = []byte(`{
  "id": "251a272b59ceefeb5bff6df5252a5805",
  "assocTxns": [
    {
      "id": "7f79dae03a46446c91623829a7fd1406",
      "type": "SECONDARY_STORAGE"
    },
    {
      "id": "71ee52c7e81044449a2babcd7bb33f7f",
      "type": "GENERATED_STORAGE"
    },
    {
      "id": "71ee52c7e81044449a2babcd7bb33f7f",
      "type": "GENERATED_STORAGE"
    },
    {
      "id": "288725b97df44c2dbd793b76f52d172c",
      "type": "GENERATED_STORAGE"
    },
    {
      "id": "288725b97df44c2dbd793b76f52d172c",
      "type": "GENERATED_STORAGE"
    }
  ],
  "responseTarget": {
    "brokers": "localhost:9092",
    "topic": "rkcy.rpg.dev.edge.GENERAL.response.0001",
    "partition": 0
  },
  "currentStepIdx": 4,
  "direction": "FORWARD",
  "uponError": "REVERT",
  "forwardSteps": [
    {
      "system": "PROCESS",
      "concern": "Character",
      "command": "ValidateCreate",
      "key": "",
      "payload": "EiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUaE0Z1bGxuYW1lXzE2NTg3ODkzODMgASoMCJYTELgPGMgqIK4q",
      "payloadDec": {
        "type": "Character",
        "instance": {
          "id": "",
          "playerId": "a431b998-c11c-485e-a946-2791d03eab6e",
          "fullname": "Fullname_1658789383",
          "active": true,
          "currency": {
            "gold": 2454,
            "faction0": 1976,
            "faction1": 5448,
            "faction2": 5422
          },
          "items": []
        }
      },
      "effectiveTime": "2021-12-13T12:29:42.486718Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "123"
      },
      "instance": "",
      "result": {
        "code": "OK",
        "processedTime": "2021-12-13T12:29:42.535509Z",
        "logEvents": [],
        "key": "",
        "payload": "EiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUaE0Z1bGxuYW1lXzE2NTg3ODkzODMgASoMCJYTELgPGMgqIK4q",
        "payloadDec": {
          "type": "Character",
          "instance": {
            "id": "",
            "playerId": "a431b998-c11c-485e-a946-2791d03eab6e",
            "fullname": "Fullname_1658789383",
            "active": true,
            "currency": {
              "gold": 2454,
              "faction0": 1976,
              "faction1": 5448,
              "faction2": 5422
            },
            "items": []
          }
        },
        "instance": "",
        "cmpdOffset": null
      }
    },
    {
      "system": "STORAGE",
      "concern": "Character",
      "command": "Create",
      "key": "",
      "payload": "EiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUaE0Z1bGxuYW1lXzE2NTg3ODkzODMgASoMCJYTELgPGMgqIK4q",
      "payloadDec": {
        "type": "Character",
        "instance": {
          "id": "",
          "playerId": "a431b998-c11c-485e-a946-2791d03eab6e",
          "fullname": "Fullname_1658789383",
          "active": true,
          "currency": {
            "gold": 2454,
            "faction0": 1976,
            "faction1": 5448,
            "faction2": 5422
          },
          "items": []
        }
      },
      "effectiveTime": "2021-12-13T12:29:42.486718Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "123"
      },
      "instance": "",
      "result": {
        "code": "OK",
        "processedTime": "2021-12-13T12:29:42.615506Z",
        "logEvents": [],
        "key": "b00d6b2a-9c6d-449d-9356-65cadc64237d",
        "payload": "cmtjeXkAAAAKJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBIkYTQzMWI5OTgtYzExYy00ODVlLWE5NDYtMjc5MWQwM2VhYjZlGhNGdWxsbmFtZV8xNjU4Nzg5MzgzIAEqDAiWExC4DxjIKiCuKg==",
        "payloadDec": {
          "type": "Character",
          "instance": {
            "id": "b00d6b2a-9c6d-449d-9356-65cadc64237d",
            "playerId": "a431b998-c11c-485e-a946-2791d03eab6e",
            "fullname": "Fullname_1658789383",
            "active": true,
            "currency": {
              "gold": 2454,
              "faction0": 1976,
              "faction1": 5448,
              "faction2": 5422
            },
            "items": []
          },
          "related": {}
        },
        "instance": "",
        "cmpdOffset": {
          "generation": 1,
          "partition": 0,
          "offset": "123"
        }
      }
    },
    {
      "system": "PROCESS",
      "concern": "Character",
      "command": "RefreshInstance",
      "key": "b00d6b2a-9c6d-449d-9356-65cadc64237d",
      "payload": "cmtjeXkAAAAKJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBIkYTQzMWI5OTgtYzExYy00ODVlLWE5NDYtMjc5MWQwM2VhYjZlGhNGdWxsbmFtZV8xNjU4Nzg5MzgzIAEqDAiWExC4DxjIKiCuKg==",
      "payloadDec": {
        "type": "Character",
        "instance": {
          "id": "b00d6b2a-9c6d-449d-9356-65cadc64237d",
          "playerId": "a431b998-c11c-485e-a946-2791d03eab6e",
          "fullname": "Fullname_1658789383",
          "active": true,
          "currency": {
            "gold": 2454,
            "faction0": 1976,
            "faction1": 5448,
            "faction2": 5422
          },
          "items": []
        },
        "related": {}
      },
      "effectiveTime": "2021-12-13T12:29:42.486718Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "124"
      },
      "instance": "",
      "result": {
        "code": "OK",
        "processedTime": "2021-12-13T12:29:42.700188Z",
        "logEvents": [],
        "key": "",
        "payload": "cmtjeXkAAAAKJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBIkYTQzMWI5OTgtYzExYy00ODVlLWE5NDYtMjc5MWQwM2VhYjZlGhNGdWxsbmFtZV8xNjU4Nzg5MzgzIAEqDAiWExC4DxjIKiCuKg==",
        "payloadDec": {
          "type": "Character",
          "instance": {
            "id": "b00d6b2a-9c6d-449d-9356-65cadc64237d",
            "playerId": "a431b998-c11c-485e-a946-2791d03eab6e",
            "fullname": "Fullname_1658789383",
            "active": true,
            "currency": {
              "gold": 2454,
              "faction0": 1976,
              "faction1": 5448,
              "faction2": 5422
            },
            "items": []
          },
          "related": {}
        },
        "instance": "",
        "cmpdOffset": null
      }
    },
    {
      "system": "PROCESS",
      "concern": "Player",
      "command": "RequestRelated",
      "key": "a431b998-c11c-485e-a946-2791d03eab6e",
      "payload": "CglDaGFyYWN0ZXISJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBoGUGxheWVyInlya2N5eQAAAAokYjAwZDZiMmEtOWM2ZC00NDlkLTkzNTYtNjVjYWRjNjQyMzdkEiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUaE0Z1bGxuYW1lXzE2NTg3ODkzODMgASoMCJYTELgPGMgqIK4q",
      "payloadDec": {
        "type": "RelatedRequest",
        "instance": {
          "concern": "Character",
          "key": "b00d6b2a-9c6d-449d-9356-65cadc64237d",
          "field": "Player",
          "payload": "cmtjeXkAAAAKJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBIkYTQzMWI5OTgtYzExYy00ODVlLWE5NDYtMjc5MWQwM2VhYjZlGhNGdWxsbmFtZV8xNjU4Nzg5MzgzIAEqDAiWExC4DxjIKiCuKg==",
          "payloadDec": {
            "type": "Character",
            "instance": {
              "id": "b00d6b2a-9c6d-449d-9356-65cadc64237d",
              "playerId": "a431b998-c11c-485e-a946-2791d03eab6e",
              "fullname": "Fullname_1658789383",
              "active": true,
              "currency": {
                "gold": 2454,
                "faction0": 1976,
                "faction1": 5448,
                "faction2": 5422
              },
              "items": []
            }
          }
        }
      },
      "effectiveTime": null,
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "118"
      },
      "instance": "",
      "result": {
        "code": "OK",
        "processedTime": "2021-12-13T12:29:42.740913Z",
        "logEvents": [],
        "key": "",
        "payload": "CgZQbGF5ZXISBlBsYXllchq0AXJrY3lBAAAACiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUSD1VzZXJfMTE3NjY5Mjk0NxgBEnEKJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBIkYTQzMWI5OTgtYzExYy00ODVlLWE5NDYtMjc5MWQwM2VhYjZlGhNGdWxsbmFtZV8xNjU4Nzg5MzgzIAEqDAiWExC4DxjIKiCuKg==",
        "payloadDec": {
          "type": "RelatedResponse",
          "instance": {
            "concern": "Player",
            "field": "Player",
            "payload": "cmtjeUEAAAAKJGE0MzFiOTk4LWMxMWMtNDg1ZS1hOTQ2LTI3OTFkMDNlYWI2ZRIPVXNlcl8xMTc2NjkyOTQ3GAEScQokYjAwZDZiMmEtOWM2ZC00NDlkLTkzNTYtNjVjYWRjNjQyMzdkEiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUaE0Z1bGxuYW1lXzE2NTg3ODkzODMgASoMCJYTELgPGMgqIK4q",
            "payloadDec": {
              "type": "Player",
              "instance": {
                "id": "a431b998-c11c-485e-a946-2791d03eab6e",
                "username": "User_1176692947",
                "active": true,
                "limitsId": ""
              },
              "related": {
                "characters": [
                  {
                    "id": "b00d6b2a-9c6d-449d-9356-65cadc64237d",
                    "playerId": "a431b998-c11c-485e-a946-2791d03eab6e",
                    "fullname": "Fullname_1658789383",
                    "active": true,
                    "currency": {
                      "gold": 2454,
                      "faction0": 1976,
                      "faction1": 5448,
                      "faction2": 5422
                    }
                  }
                ]
              }
            }
          }
        },
        "instance": "",
        "cmpdOffset": null
      }
    },
    {
      "system": "PROCESS",
      "concern": "Character",
      "command": "RefreshRelated",
      "key": "b00d6b2a-9c6d-449d-9356-65cadc64237d",
      "payload": "CgZQbGF5ZXISBlBsYXllchq0AXJrY3lBAAAACiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUSD1VzZXJfMTE3NjY5Mjk0NxgBEnEKJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBIkYTQzMWI5OTgtYzExYy00ODVlLWE5NDYtMjc5MWQwM2VhYjZlGhNGdWxsbmFtZV8xNjU4Nzg5MzgzIAEqDAiWExC4DxjIKiCuKg==",
      "payloadDec": {
        "type": "RelatedResponse",
        "instance": {
          "concern": "Player",
          "field": "Player",
          "payload": "cmtjeUEAAAAKJGE0MzFiOTk4LWMxMWMtNDg1ZS1hOTQ2LTI3OTFkMDNlYWI2ZRIPVXNlcl8xMTc2NjkyOTQ3GAEScQokYjAwZDZiMmEtOWM2ZC00NDlkLTkzNTYtNjVjYWRjNjQyMzdkEiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUaE0Z1bGxuYW1lXzE2NTg3ODkzODMgASoMCJYTELgPGMgqIK4q",
          "payloadDec": {
            "type": "Player",
            "instance": {
              "id": "a431b998-c11c-485e-a946-2791d03eab6e",
              "username": "User_1176692947",
              "active": true,
              "limitsId": ""
            },
            "related": {
              "characters": [
                {
                  "id": "b00d6b2a-9c6d-449d-9356-65cadc64237d",
                  "playerId": "a431b998-c11c-485e-a946-2791d03eab6e",
                  "fullname": "Fullname_1658789383",
                  "active": true,
                  "currency": {
                    "gold": 2454,
                    "faction0": 1976,
                    "faction1": 5448,
                    "faction2": 5422
                  }
                }
              ]
            }
          }
        }
      },
      "effectiveTime": "2021-12-13T12:29:42.741088Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "125"
      },
      "instance": "",
      "result": {
        "code": "OK",
        "processedTime": "2021-12-13T12:29:42.781459Z",
        "logEvents": [],
        "key": "",
        "payload": "cmtjeXkAAAAKJGIwMGQ2YjJhLTljNmQtNDQ5ZC05MzU2LTY1Y2FkYzY0MjM3ZBIkYTQzMWI5OTgtYzExYy00ODVlLWE5NDYtMjc5MWQwM2VhYjZlGhNGdWxsbmFtZV8xNjU4Nzg5MzgzIAEqDAiWExC4DxjIKiCuKhI5CiRhNDMxYjk5OC1jMTFjLTQ4NWUtYTk0Ni0yNzkxZDAzZWFiNmUSD1VzZXJfMTE3NjY5Mjk0NxgB",
        "payloadDec": {
          "type": "Character",
          "instance": {
            "id": "b00d6b2a-9c6d-449d-9356-65cadc64237d",
            "playerId": "a431b998-c11c-485e-a946-2791d03eab6e",
            "fullname": "Fullname_1658789383",
            "active": true,
            "currency": {
              "gold": 2454,
              "faction0": 1976,
              "faction1": 5448,
              "faction2": 5422
            },
            "items": []
          },
          "related": {
            "player": {
              "id": "a431b998-c11c-485e-a946-2791d03eab6e",
              "username": "User_1176692947",
              "active": true
            }
          }
        },
        "instance": "",
        "cmpdOffset": null
      }
    }
  ],
  "reverseSteps": []
}`)
