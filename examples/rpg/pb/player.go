// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package pb

import (
	"context"

	"github.com/jackc/pgx/v4"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

func (inst *Player) Read(ctx context.Context, key string) (*rkcy.CompoundOffset, error) {
	*inst = Player{}
	offset := &rkcy.CompoundOffset{}
	err := pool().QueryRow(ctx, "SELECT id, username, active, mro_generation, mro_partition, mro_offset FROM rpg.player WHERE id=$1", key).
		Scan(
			&inst.Id,
			&inst.Username,
			&inst.Active,
			&offset.Generation,
			&offset.Partition,
			&offset.Offset,
		)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, rkcy.NewError(rkcy.Code_NOT_FOUND, err.Error())
		}
		return nil, err
	}

	return offset, nil
}

func (inst *Player) upsert(ctx context.Context, offset *rkcy.CompoundOffset) error {
	_, err := pool().Exec(
		ctx,
		"CALL rpg.sp_upsert_player($1, $2, $3, $4, $5, $6)",
		inst.Id,
		inst.Username,
		inst.Active,
		offset.Generation,
		offset.Partition,
		offset.Offset,
	)
	return err
}

func (inst *Player) Create(ctx context.Context, offset *rkcy.CompoundOffset) error {
	return inst.upsert(ctx, offset)
}

func (inst *Player) Update(ctx context.Context, offset *rkcy.CompoundOffset) error {
	return inst.upsert(ctx, offset)
}

func (inst *Player) Delete(ctx context.Context, key string, offset *rkcy.CompoundOffset) error {
	_, err := pool().Exec(
		context.Background(),
		"CALL rpg.sp_delete_player($1, $2, $3, $4)",
		key,
		offset.Generation,
		offset.Partition,
		offset.Offset,
	)

	return err
}

func (*Player) ValidateCreate(ctx context.Context, payload *Player) (*Player, error) {
	if len(payload.Username) < 4 {
		return nil, rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Username too short")
	}
	return payload, nil
}

func (inst *Player) ValidateUpdate(ctx context.Context, payload *Player) (*Player, error) {
	if inst.Username != payload.Username {
		return nil, rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Username may not be changed")
	}
	return inst.ValidateCreate(ctx, payload)
}

func (inst *Player) SomeInterestingWork(ctx context.Context, p *Player) (*Player, error) {
	return nil, nil
}

func (inst *Player) SomeInterestingWorkVoidPayload(ctx context.Context) (*Player, error) {
	return nil, nil
}

func (inst *Player) SomeInterestingWorkVoidReturn(ctx context.Context, p *Player) error {
	return nil
}
