// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"context"

	"github.com/lachlanorr/rkcy/internal/rkcy"
	rkcy_pb "github.com/lachlanorr/rkcy/pkg/rkcy/pb"
)

type System struct {
	Name    string
	Handler func(*rkcy_pb.ApecsTxn_Step) *rkcy_pb.ApecsTxn_Step_Error
}

func Run(sys *System) {
	rkcy.Run(sys.Name)
}

func ServeGrpcGateway(ctx context.Context, srv interface{}) {
	rkcy.ServeGrpcGateway(ctx, srv)
}

func PrepLogging() {
	rkcy.PrepLogging()
}
