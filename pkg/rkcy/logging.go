// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

func LogResult(rslt *rkcypb.ApecsTxn_Step_Result, sev rkcypb.Severity, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	msg = msg[0:Mini(len(msg), 255)]

	rslt.LogEvents = append(
		rslt.LogEvents,
		&rkcypb.LogEvent{
			Sev: sev,
			Msg: msg,
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

var gLoggingIsPrepd = false

func PrepLogging() {
	if !gLoggingIsPrepd {
		gLoggingIsPrepd = true

		logLevel := os.Getenv("RKCY_LOG_LEVEL")
		if logLevel == "" {
			logLevel = "info"
		}

		badParse := false
		lvl, err := zerolog.ParseLevel(logLevel)
		if err != nil {
			lvl = zerolog.InfoLevel
			badParse = true
		}

		zerolog.SetGlobalLevel(lvl)
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02T15:04:05.999"})

		if badParse {
			log.Error().
				Msgf("Bad value for RKCY_LOG_LEVEL: %s", os.Getenv("RKCY_LOG_LEVEL"))
		}
	}
}
