// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func LogResult(rslt *rkcypb.ApecsTxn_Step_Result, sev rkcypb.Severity, format string, args ...interface{}) {
	rslt.LogEvents = append(
		rslt.LogEvents,
		&rkcypb.LogEvent{
			Sev: sev,
			Msg: fmt.Sprintf(format, args...),
		},
	)
}

func LogResultDebug(rslt *rkcypb.ApecsTxn_Step_Result, format string, args ...interface{}) {
	LogResult(rslt, rkcypb.Severity_DBG, format, args...)
}

func LogResultInfo(rslt *rkcypb.ApecsTxn_Step_Result, format string, args ...interface{}) {
	LogResult(rslt, rkcypb.Severity_INF, format, args...)
}

func LogResultWarn(rslt *rkcypb.ApecsTxn_Step_Result, format string, args ...interface{}) {
	LogResult(rslt, rkcypb.Severity_WRN, format, args...)
}

func LogResultError(rslt *rkcypb.ApecsTxn_Step_Result, format string, args ...interface{}) {
	LogResult(rslt, rkcypb.Severity_ERR, format, args...)
}

func LogProto(msg proto.Message) {
	reqJson := protojson.Format(msg)
	log.Info().Msg(reqJson)
}
