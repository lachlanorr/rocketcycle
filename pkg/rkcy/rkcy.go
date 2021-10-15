// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
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

type StandardTopicName string

const (
	ADMIN    StandardTopicName = "admin"
	PROCESS                    = "process"
	ERROR                      = "error"
	COMPLETE                   = "complete"
	STORAGE                    = "storage"
)

type PlatformImpl struct {
	Name          string
	CobraCommands []*cobra.Command

	Telem *Telemetry
}

func StartPlatform(impl *PlatformImpl) {
	runCobra(impl)
}

func InitAncillary(platformName string) {
	initPlatformName(platformName)
	prepLogging(platformName)
}

func BuildTopicNamePrefix(platformName string, concernName string, concernType Platform_Concern_Type) string {
	return fmt.Sprintf("%s.%s.%s.%s", RKCY, platformName, concernName, Platform_Concern_Type_name[int32(concernType)])
}

func BuildTopicName(topicNamePrefix string, name string, generation int32) string {
	return fmt.Sprintf("%s.%s.%04d", topicNamePrefix, name, generation)
}

func BuildFullTopicName(platformName string, concernName string, concernType Platform_Concern_Type, name string, generation int32) string {
	prefix := BuildTopicNamePrefix(platformName, concernName, concernType)
	return BuildTopicName(prefix, name, generation)
}

type TopicParts struct {
	Platform    string
	Concern     string
	Topic       string
	System      System
	ConcernType Platform_Concern_Type
	Generation  int32
}

func ParseFullTopicName(fullTopic string) (*TopicParts, error) {
	parts := strings.Split(fullTopic, ".")
	if len(parts) != 6 || parts[0] != RKCY {
		return nil, fmt.Errorf("Invalid rkcy topic: %s", fullTopic)
	}

	tp := TopicParts{
		Platform: parts[1],
		Concern:  parts[2],
		Topic:    parts[4],
	}

	if tp.Topic == PROCESS {
		tp.System = System_PROCESS
	} else if tp.Topic == STORAGE {
		tp.System = System_STORAGE
	} else {
		tp.System = System_NO_SYSTEM
	}

	concernType, ok := Platform_Concern_Type_value[parts[3]]
	if !ok {
		return nil, fmt.Errorf("Invalid rkcy topic, unable to parse ConcernType: %s", fullTopic)
	}
	tp.ConcernType = Platform_Concern_Type(concernType)

	generation, err := strconv.Atoi(parts[5])
	if err != nil {
		return nil, fmt.Errorf("Invalid rkcy topic, unable to parse Generation: %s", fullTopic)
	}
	tp.Generation = int32(generation)

	return &tp, nil
}

func (rslt *ApecsTxn_Step_Result) Log(sev Severity, format string, args ...interface{}) {
	rslt.LogEvents = append(
		rslt.LogEvents,
		&LogEvent{
			Sev: sev,
			Msg: fmt.Sprintf(format, args...),
		},
	)
}

func (rslt *ApecsTxn_Step_Result) LogDebug(format string, args ...interface{}) {
	rslt.Log(Severity_DBG, format, args...)
}

func (rslt *ApecsTxn_Step_Result) LogInfo(format string, args ...interface{}) {
	rslt.Log(Severity_INF, format, args...)
}

func (rslt *ApecsTxn_Step_Result) LogWarn(format string, args ...interface{}) {
	rslt.Log(Severity_WRN, format, args...)
}

func (rslt *ApecsTxn_Step_Result) LogError(format string, args ...interface{}) {
	rslt.Log(Severity_ERR, format, args...)
}
