// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy/consts"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
)

type Handler struct {
	Do   func(ctx context.Context, stepInfo *StepArgs) *StepResult
	Undo func(ctx context.Context, stepInfo *StepArgs) *StepResult
}

type StepArgs struct {
	ReqId         string
	ProcessedTime *timestamp.Timestamp
	Key           string
	Instance      []byte
	Payload       []byte
	Offset        *pb.Offset
}

type StepResult struct {
	Code          pb.Code
	EffectiveTime *timestamp.Timestamp
	Instance      []byte
	Payload       []byte
	LogEvents     []*pb.LogEvent
}

// These must be implemented, so we enforce this by requiring clients
// to implement this interface.
type CrudHandlers interface {
	Create(ctx context.Context, stepArgs *StepArgs) *StepResult
	Read(ctx context.Context, stepArgs *StepArgs) *StepResult
	Update(ctx context.Context, stepArgs *StepArgs) *StepResult
	Delete(ctx context.Context, stepArgs *StepArgs) *StepResult
}

type ConcernHandlers struct {
	Handlers     map[pb.Command]Handler
	CrudHandlers CrudHandlers
}

type PlatformImpl struct {
	Name          string
	CobraCommands []*cobra.Command

	// string key is ConcernName
	Handlers map[string]ConcernHandlers
}

func StartPlatform(impl *PlatformImpl) {
	runCobra(impl)
}

func InitAncillary(platformName string) {
	initPlatformName(platformName)
	prepLogging(platformName)
}

func BuildTopicNamePrefix(platformName string, concernName string, concernType pb.Platform_Concern_Type) string {
	return fmt.Sprintf("%s.%s.%s.%s", consts.Rkcy, platformName, concernName, pb.Platform_Concern_Type_name[int32(concernType)])
}

func BuildTopicName(topicNamePrefix string, name string, generation int32) string {
	return fmt.Sprintf("%s.%s.%04d", topicNamePrefix, name, generation)
}

func BuildFullTopicName(platformName string, concernName string, concernType pb.Platform_Concern_Type, name string, generation int32) string {
	prefix := BuildTopicNamePrefix(platformName, concernName, concernType)
	return fmt.Sprintf("%s.%s.%04d", prefix, name, generation)
}

type TopicParts struct {
	PlatformName string
	ConcernName  string
	TopicName    string
	System       pb.System
	ConcernType  pb.Platform_Concern_Type
	Generation   int32
}

func ParseFullTopicName(fullTopic string) (*TopicParts, error) {
	parts := strings.Split(fullTopic, ".")
	if len(parts) != 6 || parts[0] != consts.Rkcy {
		return nil, fmt.Errorf("Invalid rkcy topic: %s", fullTopic)
	}

	tp := TopicParts{
		PlatformName: parts[1],
		ConcernName:  parts[2],
		TopicName:    parts[4],
	}

	if tp.TopicName == consts.Process {
		tp.System = pb.System_PROCESS
	} else if tp.TopicName == consts.Storage {
		tp.System = pb.System_STORAGE
	} else {
		tp.System = pb.System_NO_SYSTEM
	}

	concernType, ok := pb.Platform_Concern_Type_value[parts[3]]
	if !ok {
		return nil, fmt.Errorf("Invalid rkcy topic, unable to parse ConcernType: %s", fullTopic)
	}
	tp.ConcernType = pb.Platform_Concern_Type(concernType)

	generation, err := strconv.Atoi(parts[5])
	if err != nil {
		return nil, fmt.Errorf("Invalid rkcy topic, unable to parse Generation: %s", fullTopic)
	}
	tp.Generation = int32(generation)

	return &tp, nil
}

func (rslt *StepResult) Log(sev pb.Severity, format string, args ...interface{}) {
	rslt.LogEvents = append(
		rslt.LogEvents,
		&pb.LogEvent{
			Sev: sev,
			Msg: fmt.Sprintf(format, args...),
		},
	)
}

func (rslt *StepResult) LogDebug(format string, args ...interface{}) {
	rslt.Log(pb.Severity_DEBUG, format, args...)
}

func (rslt *StepResult) LogInfo(format string, args ...interface{}) {
	rslt.Log(pb.Severity_INFO, format, args...)
}

func (rslt *StepResult) LogWarn(format string, args ...interface{}) {
	rslt.Log(pb.Severity_WARN, format, args...)
}

func (rslt *StepResult) LogError(format string, args ...interface{}) {
	rslt.Log(pb.Severity_ERROR, format, args...)
}
