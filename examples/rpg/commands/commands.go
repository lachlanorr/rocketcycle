// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package commands

import (
	"fmt"

	"github.com/lachlanorr/rkcy/examples/rpg/codes"
	rkcy_pb "github.com/lachlanorr/rkcy/pkg/rkcy/pb"
)

const (
	Create = 1
)

func Handler(step *rkcy_pb.ApecsTxn_Step) *rkcy_pb.ApecsTxn_Step_Error {
	switch step.Command {
	case Create:
		fmt.Printf("Create %s\n", step.AppName)
		return nil
	}

	return &rkcy_pb.ApecsTxn_Step_Error{
		Code: codes.UnknownCommand,
	}
}
