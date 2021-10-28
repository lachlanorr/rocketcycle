// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"bytes"
	"context"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy/jsonutils"

	_ "github.com/lachlanorr/rocketcycle/examples/rpg/logic"
	_ "github.com/lachlanorr/rocketcycle/examples/rpg/pb"
)

func TestMarshalTxn(t *testing.T) {
	// get a protobuf from our sample json so we have a good start
	txn := &rkcy.ApecsTxn{}
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
	txn := &rkcy.ApecsTxn{}
	err := protojson.Unmarshal(testTxnJson, txn)
	if err != nil {
		t.Fatal(err)
	}

	pjOpts := protojson.MarshalOptions{Multiline: true, Indent: "  ", EmitUnpopulated: true}
	marTxnJson, err := rkcy.DecodeTxnOpaques(ctx, txn, &pjOpts)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(marTxnJson, testTxnJsonDec) {
		t.Errorf("Marshalled bytes do not match expected:\n\t%s\n\t-- vs.\n\t%s", string(marTxnJson), string(testTxnJson))
	}
}

var testTxnJson = []byte(`{
  "id": "a521d9c32d92af0d5f27104e2b6be121",
  "assocTxns": [],
  "responseTarget": {
    "brokers": "localhost:9092",
    "topic": "rkcy.rpg.edge.GENERAL.response.0001",
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
      "payload": "EiQwZTU4NzA2Yi1kYWNhLTQ0MzctYjFjNi0zNzZiOGQ1N2QyZjIaE0Z1bGxuYW1lXzEzOTcxMDYxMzEgASoMCOtGEIs4GOo9IPBC",
      "effectiveTime": "2021-10-26T18:16:15.576872Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "0"
      },
      "result": {
        "code": "OK",
        "processedTime": "2021-10-26T18:16:15.660973Z",
        "logEvents": [],
        "key": "",
        "payload": "EiQwZTU4NzA2Yi1kYWNhLTQ0MzctYjFjNi0zNzZiOGQ1N2QyZjIaE0Z1bGxuYW1lXzEzOTcxMDYxMzEgASoMCOtGEIs4GOo9IPBC",
        "instance": "",
        "cmpdOffset": null
      }
    },
    {
      "system": "STORAGE",
      "concern": "Character",
      "command": "Create",
      "key": "",
      "payload": "EiQwZTU4NzA2Yi1kYWNhLTQ0MzctYjFjNi0zNzZiOGQ1N2QyZjIaE0Z1bGxuYW1lXzEzOTcxMDYxMzEgASoMCOtGEIs4GOo9IPBC",
      "effectiveTime": "2021-10-26T18:16:15.576872Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "0"
      },
      "result": {
        "code": "OK",
        "processedTime": "2021-10-26T18:16:15.747558Z",
        "logEvents": [],
        "key": "8b65eb9c-1454-4547-8a4b-bde3cd45485c",
        "payload": "CiQ4YjY1ZWI5Yy0xNDU0LTQ1NDctOGE0Yi1iZGUzY2Q0NTQ4NWMSJDBlNTg3MDZiLWRhY2EtNDQzNy1iMWM2LTM3NmI4ZDU3ZDJmMhoTRnVsbG5hbWVfMTM5NzEwNjEzMSABKgwI60YQizgY6j0g8EI=",
        "instance": "",
        "cmpdOffset": {
          "generation": 1,
          "partition": 0,
          "offset": "0"
        }
      }
    },
    {
      "system": "PROCESS",
      "concern": "Character",
      "command": "RefreshInstance",
      "key": "8b65eb9c-1454-4547-8a4b-bde3cd45485c",
      "payload": "cmtjeXkAAAAKJDhiNjVlYjljLTE0NTQtNDU0Ny04YTRiLWJkZTNjZDQ1NDg1YxIkMGU1ODcwNmItZGFjYS00NDM3LWIxYzYtMzc2YjhkNTdkMmYyGhNGdWxsbmFtZV8xMzk3MTA2MTMxIAEqDAjrRhCLOBjqPSDwQg==",
      "effectiveTime": "2021-10-26T18:16:15.576872Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "1"
      },
      "result": {
        "code": "OK",
        "processedTime": "2021-10-26T18:16:15.875846Z",
        "logEvents": [],
        "key": "",
        "payload": "cmtjeXkAAAAKJDhiNjVlYjljLTE0NTQtNDU0Ny04YTRiLWJkZTNjZDQ1NDg1YxIkMGU1ODcwNmItZGFjYS00NDM3LWIxYzYtMzc2YjhkNTdkMmYyGhNGdWxsbmFtZV8xMzk3MTA2MTMxIAEqDAjrRhCLOBjqPSDwQg==",
        "instance": "",
        "cmpdOffset": null
      }
    },
    {
      "system": "PROCESS",
      "concern": "Player",
      "command": "RequestRelated",
      "key": "0e58706b-daca-4437-b1c6-376b8d57d2f2",
      "payload": "CglDaGFyYWN0ZXISJDhiNjVlYjljLTE0NTQtNDU0Ny04YTRiLWJkZTNjZDQ1NDg1YxoIUGxheWVySWQicQokOGI2NWViOWMtMTQ1NC00NTQ3LThhNGItYmRlM2NkNDU0ODVjEiQwZTU4NzA2Yi1kYWNhLTQ0MzctYjFjNi0zNzZiOGQ1N2QyZjIaE0Z1bGxuYW1lXzEzOTcxMDYxMzEgASoMCOtGEIs4GOo9IPBC",
      "effectiveTime": null,
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "2"
      },
      "result": {
        "code": "OK",
        "processedTime": "2021-10-26T18:16:15.923031Z",
        "logEvents": [],
        "key": "",
        "payload": "CgZQbGF5ZXISCFBsYXllcklkGjgKJDBlNTg3MDZiLWRhY2EtNDQzNy1iMWM2LTM3NmI4ZDU3ZDJmMhIOVXNlcl83MjE3NDkxMjEYAQ==",
        "instance": "",
        "cmpdOffset": null
      }
    },
    {
      "system": "PROCESS",
      "concern": "Character",
      "command": "RefreshRelated",
      "key": "8b65eb9c-1454-4547-8a4b-bde3cd45485c",
      "payload": "CgZQbGF5ZXISCFBsYXllcklkGjgKJDBlNTg3MDZiLWRhY2EtNDQzNy1iMWM2LTM3NmI4ZDU3ZDJmMhIOVXNlcl83MjE3NDkxMjEYAQ==",
      "effectiveTime": "2021-10-26T18:16:15.923486Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "2"
      },
      "result": {
        "code": "OK",
        "processedTime": "2021-10-26T18:16:15.989488Z",
        "logEvents": [],
        "key": "",
        "payload": "cmtjeXkAAAAKJDhiNjVlYjljLTE0NTQtNDU0Ny04YTRiLWJkZTNjZDQ1NDg1YxIkMGU1ODcwNmItZGFjYS00NDM3LWIxYzYtMzc2YjhkNTdkMmYyGhNGdWxsbmFtZV8xMzk3MTA2MTMxIAEqDAjrRhCLOBjqPSDwQgo4CiQwZTU4NzA2Yi1kYWNhLTQ0MzctYjFjNi0zNzZiOGQ1N2QyZjISDlVzZXJfNzIxNzQ5MTIxGAE=",
        "instance": "",
        "cmpdOffset": null
      }
    }
  ],
  "reverseSteps": []
}`)

var testTxnJsonDec = []byte(`{
  "id": "a521d9c32d92af0d5f27104e2b6be121",
  "assocTxns": [],
  "responseTarget": {
    "brokers": "localhost:9092",
    "topic": "rkcy.rpg.edge.GENERAL.response.0001",
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
      "payload": "EiQwZTU4NzA2Yi1kYWNhLTQ0MzctYjFjNi0zNzZiOGQ1N2QyZjIaE0Z1bGxuYW1lXzEzOTcxMDYxMzEgASoMCOtGEIs4GOo9IPBC",
      "payloadDec": {
        "type": "Character",
        "instance": {
          "id": "",
          "playerId": "0e58706b-daca-4437-b1c6-376b8d57d2f2",
          "fullname": "Fullname_1397106131",
          "active": true,
          "currency": {
            "gold": 9067,
            "faction0": 7179,
            "faction1": 7914,
            "faction2": 8560
          },
          "items": []
        }
      },
      "effectiveTime": "2021-10-26T18:16:15.576872Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "0"
      },
      "result": {
        "code": "OK",
        "processedTime": "2021-10-26T18:16:15.660973Z",
        "logEvents": [],
        "key": "",
        "payload": "EiQwZTU4NzA2Yi1kYWNhLTQ0MzctYjFjNi0zNzZiOGQ1N2QyZjIaE0Z1bGxuYW1lXzEzOTcxMDYxMzEgASoMCOtGEIs4GOo9IPBC",
        "payloadDec": {
          "type": "Character",
          "instance": {
            "id": "",
            "playerId": "0e58706b-daca-4437-b1c6-376b8d57d2f2",
            "fullname": "Fullname_1397106131",
            "active": true,
            "currency": {
              "gold": 9067,
              "faction0": 7179,
              "faction1": 7914,
              "faction2": 8560
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
      "payload": "EiQwZTU4NzA2Yi1kYWNhLTQ0MzctYjFjNi0zNzZiOGQ1N2QyZjIaE0Z1bGxuYW1lXzEzOTcxMDYxMzEgASoMCOtGEIs4GOo9IPBC",
      "payloadDec": {
        "type": "Character",
        "instance": {
          "id": "",
          "playerId": "0e58706b-daca-4437-b1c6-376b8d57d2f2",
          "fullname": "Fullname_1397106131",
          "active": true,
          "currency": {
            "gold": 9067,
            "faction0": 7179,
            "faction1": 7914,
            "faction2": 8560
          },
          "items": []
        }
      },
      "effectiveTime": "2021-10-26T18:16:15.576872Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "0"
      },
      "result": {
        "code": "OK",
        "processedTime": "2021-10-26T18:16:15.747558Z",
        "logEvents": [],
        "key": "8b65eb9c-1454-4547-8a4b-bde3cd45485c",
        "payload": "CiQ4YjY1ZWI5Yy0xNDU0LTQ1NDctOGE0Yi1iZGUzY2Q0NTQ4NWMSJDBlNTg3MDZiLWRhY2EtNDQzNy1iMWM2LTM3NmI4ZDU3ZDJmMhoTRnVsbG5hbWVfMTM5NzEwNjEzMSABKgwI60YQizgY6j0g8EI=",
        "payloadDec": {
          "type": "Character",
          "instance": {
            "id": "8b65eb9c-1454-4547-8a4b-bde3cd45485c",
            "playerId": "0e58706b-daca-4437-b1c6-376b8d57d2f2",
            "fullname": "Fullname_1397106131",
            "active": true,
            "currency": {
              "gold": 9067,
              "faction0": 7179,
              "faction1": 7914,
              "faction2": 8560
            },
            "items": []
          }
        },
        "instance": "",
        "cmpdOffset": {
          "generation": 1,
          "partition": 0,
          "offset": "0"
        }
      }
    },
    {
      "system": "PROCESS",
      "concern": "Character",
      "command": "RefreshInstance",
      "key": "8b65eb9c-1454-4547-8a4b-bde3cd45485c",
      "payload": "cmtjeXkAAAAKJDhiNjVlYjljLTE0NTQtNDU0Ny04YTRiLWJkZTNjZDQ1NDg1YxIkMGU1ODcwNmItZGFjYS00NDM3LWIxYzYtMzc2YjhkNTdkMmYyGhNGdWxsbmFtZV8xMzk3MTA2MTMxIAEqDAjrRhCLOBjqPSDwQg==",
      "payloadDec": {
        "type": "Character",
        "instance": {
          "id": "8b65eb9c-1454-4547-8a4b-bde3cd45485c",
          "playerId": "0e58706b-daca-4437-b1c6-376b8d57d2f2",
          "fullname": "Fullname_1397106131",
          "active": true,
          "currency": {
            "gold": 9067,
            "faction0": 7179,
            "faction1": 7914,
            "faction2": 8560
          },
          "items": []
        }
      },
      "effectiveTime": "2021-10-26T18:16:15.576872Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "1"
      },
      "result": {
        "code": "OK",
        "processedTime": "2021-10-26T18:16:15.875846Z",
        "logEvents": [],
        "key": "",
        "payload": "cmtjeXkAAAAKJDhiNjVlYjljLTE0NTQtNDU0Ny04YTRiLWJkZTNjZDQ1NDg1YxIkMGU1ODcwNmItZGFjYS00NDM3LWIxYzYtMzc2YjhkNTdkMmYyGhNGdWxsbmFtZV8xMzk3MTA2MTMxIAEqDAjrRhCLOBjqPSDwQg==",
        "payloadDec": {
          "type": "Character",
          "instance": {
            "id": "8b65eb9c-1454-4547-8a4b-bde3cd45485c",
            "playerId": "0e58706b-daca-4437-b1c6-376b8d57d2f2",
            "fullname": "Fullname_1397106131",
            "active": true,
            "currency": {
              "gold": 9067,
              "faction0": 7179,
              "faction1": 7914,
              "faction2": 8560
            },
            "items": []
          }
        },
        "instance": "",
        "cmpdOffset": null
      }
    },
    {
      "system": "PROCESS",
      "concern": "Player",
      "command": "RequestRelated",
      "key": "0e58706b-daca-4437-b1c6-376b8d57d2f2",
      "payload": "CglDaGFyYWN0ZXISJDhiNjVlYjljLTE0NTQtNDU0Ny04YTRiLWJkZTNjZDQ1NDg1YxoIUGxheWVySWQicQokOGI2NWViOWMtMTQ1NC00NTQ3LThhNGItYmRlM2NkNDU0ODVjEiQwZTU4NzA2Yi1kYWNhLTQ0MzctYjFjNi0zNzZiOGQ1N2QyZjIaE0Z1bGxuYW1lXzEzOTcxMDYxMzEgASoMCOtGEIs4GOo9IPBC",
      "payloadDec": {
        "type": "RelatedRequest",
        "instance": {
          "concern": "Character",
          "key": "8b65eb9c-1454-4547-8a4b-bde3cd45485c",
          "field": "PlayerId",
          "payload": "CiQ4YjY1ZWI5Yy0xNDU0LTQ1NDctOGE0Yi1iZGUzY2Q0NTQ4NWMSJDBlNTg3MDZiLWRhY2EtNDQzNy1iMWM2LTM3NmI4ZDU3ZDJmMhoTRnVsbG5hbWVfMTM5NzEwNjEzMSABKgwI60YQizgY6j0g8EI="
        }
      },
      "effectiveTime": null,
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "2"
      },
      "result": {
        "code": "OK",
        "processedTime": "2021-10-26T18:16:15.923031Z",
        "logEvents": [],
        "key": "",
        "payload": "CgZQbGF5ZXISCFBsYXllcklkGjgKJDBlNTg3MDZiLWRhY2EtNDQzNy1iMWM2LTM3NmI4ZDU3ZDJmMhIOVXNlcl83MjE3NDkxMjEYAQ==",
        "payloadDec": {
          "type": "RelatedResponse",
          "instance": {
            "concern": "Player",
            "field": "PlayerId",
            "payload": "CiQwZTU4NzA2Yi1kYWNhLTQ0MzctYjFjNi0zNzZiOGQ1N2QyZjISDlVzZXJfNzIxNzQ5MTIxGAE="
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
      "key": "8b65eb9c-1454-4547-8a4b-bde3cd45485c",
      "payload": "CgZQbGF5ZXISCFBsYXllcklkGjgKJDBlNTg3MDZiLWRhY2EtNDQzNy1iMWM2LTM3NmI4ZDU3ZDJmMhIOVXNlcl83MjE3NDkxMjEYAQ==",
      "payloadDec": {
        "type": "RelatedResponse",
        "instance": {
          "concern": "Player",
          "field": "PlayerId",
          "payload": "CiQwZTU4NzA2Yi1kYWNhLTQ0MzctYjFjNi0zNzZiOGQ1N2QyZjISDlVzZXJfNzIxNzQ5MTIxGAE="
        }
      },
      "effectiveTime": "2021-10-26T18:16:15.923486Z",
      "cmpdOffset": {
        "generation": 1,
        "partition": 0,
        "offset": "2"
      },
      "result": {
        "code": "OK",
        "processedTime": "2021-10-26T18:16:15.989488Z",
        "logEvents": [],
        "key": "",
        "payload": "cmtjeXkAAAAKJDhiNjVlYjljLTE0NTQtNDU0Ny04YTRiLWJkZTNjZDQ1NDg1YxIkMGU1ODcwNmItZGFjYS00NDM3LWIxYzYtMzc2YjhkNTdkMmYyGhNGdWxsbmFtZV8xMzk3MTA2MTMxIAEqDAjrRhCLOBjqPSDwQgo4CiQwZTU4NzA2Yi1kYWNhLTQ0MzctYjFjNi0zNzZiOGQ1N2QyZjISDlVzZXJfNzIxNzQ5MTIxGAE=",
        "payloadDec": {
          "type": "Character",
          "instance": {
            "id": "8b65eb9c-1454-4547-8a4b-bde3cd45485c",
            "playerId": "0e58706b-daca-4437-b1c6-376b8d57d2f2",
            "fullname": "Fullname_1397106131",
            "active": true,
            "currency": {
              "gold": 9067,
              "faction0": 7179,
              "faction1": 7914,
              "faction2": 8560
            },
            "items": []
          },
          "related": {
            "player": {
              "id": "0e58706b-daca-4437-b1c6-376b8d57d2f2",
              "username": "User_721749121",
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
