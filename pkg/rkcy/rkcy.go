// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"fmt"
	"time"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

const (
	RKCY string = "rkcy"

	DIRECTIVE_HEADER    string = "rkcy-directive"
	TRACE_PARENT_HEADER string = "traceparent"

	RKCY_TOPIC     string = "rkcy.topic"
	RKCY_PARTITION string = "rkcy.partition"

	MAX_PARTITION                  int32 = 1024
	PLATFORM_TOPIC_RETENTION_BYTES int32 = 10 * 1024 * 1024
)

type PlatformArgs struct {
	Platform          string
	Environment       string
	AdminBrokers      string
	AdminPingInterval time.Duration
}

type Error struct {
	Code rkcypb.Code
	Msg  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", rkcypb.Code_name[int32(e.Code)], e.Msg)
}

func NewError(code rkcypb.Code, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}
