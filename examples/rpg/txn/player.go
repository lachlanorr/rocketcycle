// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package txn

import (
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"

	"github.com/lachlanorr/rocketcycle/examples/rpg/pb"
)

func ReadPlayer(id string) *rkcy.Txn {
	return &rkcy.Txn{
		Revert: rkcy.NonRevertable,
		Steps: []rkcy.Step{
			{
				Concern: "Player",
				Command: rkcy.READ,
				Key:     id,
			},
		},
	}
}

func CreatePlayer(inst *pb.Player) *rkcy.Txn {
	return &rkcy.Txn{
		Revert: rkcy.Revertable,
		Steps: []rkcy.Step{
			{
				Concern: "Player",
				Command: rkcy.CREATE,
				Payload: inst,
			},
		},
	}
}

func UpdatePlayer(inst *pb.Player) *rkcy.Txn {
	return &rkcy.Txn{
		Revert: rkcy.Revertable,
		Steps: []rkcy.Step{
			{
				Concern: "Player",
				Command: rkcy.UPDATE,
				Payload: inst,
			},
		},
	}
}

func DeletePlayer(id string) *rkcy.Txn {
	return &rkcy.Txn{
		Revert: rkcy.Revertable,
		Steps: []rkcy.Step{
			{
				Concern: "Player",
				Command: rkcy.DELETE,
				Key:     id,
			},
		},
	}
}
