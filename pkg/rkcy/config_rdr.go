// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type ConfigRdr interface {
	GetString(key string) (string, bool)
	GetBool(key string) (bool, bool)
	GetFloat64(key string) (float64, bool)
	GetComplexMsg(msgType string, key string) (proto.Message, bool)
	GetComplexBytes(msgType string, key string) ([]byte, bool)

	BuildConfigResponse() *rkcypb.ConfigReadResponse
}
