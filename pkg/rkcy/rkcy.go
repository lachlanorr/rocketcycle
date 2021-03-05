// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"fmt"

	timestamp "github.com/golang/protobuf/ptypes/timestamp"
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
	Name     string
	Handlers map[string]*ConcernHandlers
}

func StartPlatform(impl *PlatformImpl) {
	prepLogging(impl.Name)
	runCobra(impl.Name)
}

func InitAncillary(platformName string) {
	prepLogging(platformName)
	initPlatformName(platformName)
}

func BuildTopicNamePrefix(platformName string, concern string, concernType pb.Platform_Concern_Type) string {
	return fmt.Sprintf("rkcy.%s.%s.%s", platformName, concern, pb.Platform_Concern_Type_name[int32(concernType)])
}

func BuildTopicName(topicNamePrefix string, name string, generation int32) string {
	return fmt.Sprintf("%s.%s.%04d", topicNamePrefix, name, generation)
}
