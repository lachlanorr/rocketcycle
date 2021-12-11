// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

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

type StandardTopicName string

const (
	ADMIN        StandardTopicName = "admin"
	PROCESS                        = "process"
	ERROR                          = "error"
	COMPLETE                       = "complete"
	STORAGE                        = "storage"
	STORAGE_SCND                   = "storage-scnd"
)

var nameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z\-]{1,15}$`)

func IsValidName(name string) bool {
	return nameRe.MatchString(name)
}

func PlatformTopic(platformName string, environment string) string {
	return fmt.Sprintf("%s.%s.%s.platform", RKCY, platformName, environment)
}

func ConfigTopic(platformName string, environment string) string {
	return fmt.Sprintf("%s.%s.%s.config", RKCY, platformName, environment)
}

func ProducersTopic(platformName string, environment string) string {
	return fmt.Sprintf("%s.%s.%s.producers", RKCY, platformName, environment)
}

func ConsumersTopic(platformName string, environment string) string {
	return fmt.Sprintf("%s.%s.%s.consumers", RKCY, platformName, environment)
}

func BuildTopicNamePrefix(platformName string, environment string, concernName string, concernType rkcypb.Concern_Type) string {
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

func BuildFullTopicName(platformName string, environment string, concernName string, concernType rkcypb.Concern_Type, name string, generation int32) string {
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
	System      rkcypb.System
	ConcernType rkcypb.Concern_Type
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
		tp.System = rkcypb.System_PROCESS
	} else if tp.Topic == STORAGE {
		tp.System = rkcypb.System_STORAGE
	} else if tp.Topic == STORAGE_SCND {
		tp.System = rkcypb.System_STORAGE_SCND
	} else {
		tp.System = rkcypb.System_NO_SYSTEM
	}

	concernType, ok := rkcypb.Concern_Type_value[parts[4]]
	if !ok {
		return nil, fmt.Errorf("Invalid rkcy topic, unable to parse ConcernType: %s", fullTopic)
	}
	tp.ConcernType = rkcypb.Concern_Type(concernType)

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

func SetStepResult(rslt *rkcypb.ApecsTxn_Step_Result, err error) {
	if err == nil {
		rslt.Code = rkcypb.Code_OK
	} else {
		rkcyErr, ok := err.(*Error)
		if ok {
			rslt.Code = rkcyErr.Code
			rslt.LogEvents = append(rslt.LogEvents, &rkcypb.LogEvent{Sev: rkcypb.Severity_ERR, Msg: rkcyErr.Msg})
		} else {
			rslt.Code = rkcypb.Code_INTERNAL
			rslt.LogEvents = append(rslt.LogEvents, &rkcypb.LogEvent{Sev: rkcypb.Severity_ERR, Msg: err.Error()})
		}
	}
}
