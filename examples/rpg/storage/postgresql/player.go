// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package postgresql

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"

	"github.com/lachlanorr/rocketcycle/examples/rpg/pb"
)

func init() {
	rkcy.RegisterStorageCommands("postgresql", "Player", &Player{})
}

type Player struct{}

func (*Player) Read(ctx context.Context, key string) (*pb.Player, *pb.PlayerRelatedConcerns, *rkcy.CompoundOffset, error) {
	inst := &pb.Player{}
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
			return nil, nil, nil, rkcy.NewError(rkcy.Code_NOT_FOUND, err.Error())
		}
		return nil, nil, nil, err
	}

	return inst, nil, offset, nil
}

func (p *Player) Create(ctx context.Context, inst *pb.Player, cmpdOffset *rkcy.CompoundOffset) (*pb.Player, error) {
	inst.Id = uuid.NewString()
	err := p.upsert(ctx, inst, nil, cmpdOffset)
	if err != nil {
		return nil, err
	}
	return inst, nil
}

func (p *Player) Update(ctx context.Context, inst *pb.Player, relCnc *pb.PlayerRelatedConcerns, cmpdOffset *rkcy.CompoundOffset) error {
	return p.upsert(ctx, inst, relCnc, cmpdOffset)
}

func (*Player) Delete(ctx context.Context, key string, cmpdOffset *rkcy.CompoundOffset) error {
	_, err := pool().Exec(
		context.Background(),
		"CALL rpg.sp_delete_player($1, $2, $3, $4)",
		key,
		cmpdOffset.Generation,
		cmpdOffset.Partition,
		cmpdOffset.Offset,
	)
	return err
}

func (*Player) upsert(ctx context.Context, inst *pb.Player, relCnc *pb.PlayerRelatedConcerns, offset *rkcy.CompoundOffset) error {
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
