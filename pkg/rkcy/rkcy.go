// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"

	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/lachlanorr/rkcy/internal/rkcy"
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

func Start(handlers *PlatformImpl) {
	InitPlatformName(handlers.Name)
}

func InitPlatformName(platformName string) {
	rkcy.InitPlatformName(platformName)
	rkcy.PrepLogging()
}

func ServeGrpcGateway(ctx context.Context, srv interface{}) {
	rkcy.ServeGrpcGateway(ctx, srv)
}
