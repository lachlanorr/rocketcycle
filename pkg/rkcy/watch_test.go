// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"fmt"
	"testing"

	//	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestDecodeTxnOpaques(t *testing.T) {
	txn1 := &ApecsTxn{}
	err := protojson.Unmarshal([]byte(testTxn1Json), txn1)
	if err != nil {
		t.Error(err)
	}

}

var testTxn1Json = `{
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
}`
