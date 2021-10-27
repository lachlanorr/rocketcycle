// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package txn

import (
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"

	"github.com/lachlanorr/rocketcycle/examples/rpg/consts"
	"github.com/lachlanorr/rocketcycle/examples/rpg/pb"
)

func Fund(fr *pb.FundingRequest) *rkcy.Txn {
	return &rkcy.Txn{
		Revert: rkcy.Revertable,
		Steps: []rkcy.Step{
			{
				Concern: consts.Character,
				Command: "Fund",
				Key:     fr.CharacterId,
				Payload: fr,
			},
		},
	}
}

func Trade(tr *pb.TradeRequest) *rkcy.Txn {
	return &rkcy.Txn{
		Revert: rkcy.Revertable,
		Steps: []rkcy.Step{
			{
				Concern: "Character",
				Command: "DebitFunds",
				Key:     tr.Lhs.CharacterId,
				Payload: tr.Lhs,
			},
			{
				Concern: "Character",
				Command: "DebitFunds",
				Key:     tr.Rhs.CharacterId,
				Payload: tr.Rhs,
			},
			{
				Concern: "Character",
				Command: "CreditFunds",
				Key:     tr.Lhs.CharacterId,
				Payload: tr.Rhs,
			},
			{
				Concern: "Character",
				Command: "CreditFunds",
				Key:     tr.Rhs.CharacterId,
				Payload: tr.Lhs,
			},
		},
	}
}

func ReadCharacter(id string) *rkcy.Txn {
	return &rkcy.Txn{
		Revert: rkcy.NonRevertable,
		Steps: []rkcy.Step{
			{
				Concern: "Character",
				Command: rkcy.READ,
				Key:     id,
			},
		},
	}
}

func CreateCharacter(inst *pb.Character) *rkcy.Txn {
	return &rkcy.Txn{
		Revert: rkcy.Revertable,
		Steps: []rkcy.Step{
			{
				Concern: "Character",
				Command: rkcy.CREATE,
				Payload: inst,
			},
		},
	}
}

func UpdateCharacter(inst *pb.Character) *rkcy.Txn {
	return &rkcy.Txn{
		Revert: rkcy.Revertable,
		Steps: []rkcy.Step{
			{
				Concern: "Character",
				Command: rkcy.UPDATE,
				Payload: inst,
			},
		},
	}
}

func DeleteCharacter(id string) *rkcy.Txn {
	return &rkcy.Txn{
		Revert: rkcy.Revertable,
		Steps: []rkcy.Step{
			{
				Concern: "Character",
				Command: rkcy.DELETE,
				Key:     id,
			},
		},
	}
}
