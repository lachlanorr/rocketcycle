// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"fmt"

	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rkcy/pkg/rkcy/pb"
)

type ConcernHandlerResult struct {
	Status        pb.ApecsTxn_Step_Status
	EffectiveTime *timestamp.Timestamp
	LogEvents     []pb.ApecsTxn_Step_LogEvent
}

type ConcernHandlers struct {
	ConcernName string
	Handlers    map[int32]func(payload []byte) *ConcernHandlerResult
}

type PlatformImpl struct {
	Name          string
	CobraCommands []*cobra.Command
	Handlers      map[string]*ConcernHandlers
}

func StartPlatform(impl *PlatformImpl) {
	runCobra(impl)
}

func InitAncillary(platformName string) {
	initPlatformName(platformName)
	prepLogging(platformName)
}

func BuildTopicNamePrefix(platformName string, concernName string, concernType pb.Platform_Concern_Type) string {
	return fmt.Sprintf("%s.%s.%s.%s", rkcy, platformName, concernName, pb.Platform_Concern_Type_name[int32(concernType)])
}

func BuildTopicName(topicNamePrefix string, name string, generation int32) string {
	return fmt.Sprintf("%s.%s.%04d", topicNamePrefix, name, generation)
}

func BuildFullTopicName(platformName string, concernName string, concernType pb.Platform_Concern_Type, name string, generation int32) string {
	prefix := BuildTopicNamePrefix(platformName, concernName, concernType)
	return fmt.Sprintf("%s.%s.%04d", prefix, name, generation)
}
