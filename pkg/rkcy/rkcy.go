// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
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
	ADMIN        StandardTopicName = "admin"
	PROCESS                        = "process"
	ERROR                          = "error"
	COMPLETE                       = "complete"
	STORAGE                        = "storage"
	STORAGE_SCND                   = "storage-scnd"
)

type StorageInit func(ctx context.Context, config map[string]string, wg *sync.WaitGroup) error

func BuildTopicNamePrefix(platformName string, environment string, concernName string, concernType Concern_Type) string {
	if !IsValidName(platformName) {
		log.Fatal().Msgf("Invalid platformName: %s", platformName)
	}
	if !IsValidName(environment) {
		log.Fatal().Msgf("Invalid environment: %s", environment)
	}
	if !IsValidName(concernName) {
		log.Fatal().Msgf("Invalid concernName: %s", concernName)
	}
	if !IsValidName(concernType.String()) {
		log.Fatal().Msgf("Invalid concernType: %s", concernType.String())
	}
	return fmt.Sprintf("%s.%s.%s.%s.%s", RKCY, platformName, environment, concernName, concernType.String())
}

func BuildTopicName(topicNamePrefix string, name string, generation int32) string {
	if !IsValidName(name) {
		log.Fatal().Msgf("Invalid topicName: %s", name)
	}
	if generation <= 0 {
		log.Fatal().Msgf("Invalid generation: %d", generation)
	}
	return fmt.Sprintf("%s.%s.%04d", topicNamePrefix, name, generation)
}

func BuildFullTopicName(platformName string, environment string, concernName string, concernType Concern_Type, name string, generation int32) string {
	if !IsValidName(platformName) {
		log.Fatal().Msgf("Invalid platformName: %s", platformName)
	}
	if !IsValidName(environment) {
		log.Fatal().Msgf("Invalid environment: %s", environment)
	}
	if !IsValidName(concernName) {
		log.Fatal().Msgf("Invalid concernName: %s", concernName)
	}
	if !IsValidName(concernType.String()) {
		log.Fatal().Msgf("Invalid concernType: %s", concernType.String())
	}
	if !IsValidName(name) {
		log.Fatal().Msgf("Invalid topicName: %s", name)
	}
	if generation <= 0 {
		log.Fatal().Msgf("Invalid generation: %d", generation)
	}
	prefix := BuildTopicNamePrefix(platformName, environment, concernName, concernType)
	return BuildTopicName(prefix, name, generation)
}

type TopicParts struct {
	Platform    string
	Environment string
	Concern     string
	Topic       string
	System      System
	ConcernType Concern_Type
	Generation  int32
}

func ParseFullTopicName(fullTopic string) (*TopicParts, error) {
	parts := strings.Split(fullTopic, ".")
	if len(parts) != 7 || parts[0] != RKCY {
		return nil, fmt.Errorf("Invalid rkcy topic: %s", fullTopic)
	}

	tp := TopicParts{
		Platform:    parts[1],
		Environment: parts[2],
		Concern:     parts[3],
		Topic:       parts[5],
	}

	if !IsValidName(tp.Platform) {
		return nil, fmt.Errorf("Invalid tp.Platform: %s", tp.Platform)
	}
	if !IsValidName(tp.Environment) {
		return nil, fmt.Errorf("Invalid tp.Environment: %s", tp.Environment)
	}
	if !IsValidName(tp.Concern) {
		return nil, fmt.Errorf("Invalid tp.Concern: %s", tp.Concern)
	}
	if !IsValidName(tp.Topic) {
		return nil, fmt.Errorf("Invalid tp.Topic: %s", tp.Topic)
	}

	if tp.Topic == PROCESS {
		tp.System = System_PROCESS
	} else if tp.Topic == STORAGE {
		tp.System = System_STORAGE
	} else if tp.Topic == STORAGE_SCND {
		tp.System = System_STORAGE_SCND
	} else {
		tp.System = System_NO_SYSTEM
	}

	concernType, ok := Concern_Type_value[parts[4]]
	if !ok {
		return nil, fmt.Errorf("Invalid rkcy topic, unable to parse ConcernType: %s", fullTopic)
	}
	tp.ConcernType = Concern_Type(concernType)

	generation, err := strconv.Atoi(parts[6])
	if err != nil {
		return nil, fmt.Errorf("Invalid rkcy topic, unable to parse Generation: %s", fullTopic)
	}
	tp.Generation = int32(generation)
	if tp.Generation < 0 {
		return nil, fmt.Errorf("Invalid tp.Generation: %d", tp.Generation)
	}

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
