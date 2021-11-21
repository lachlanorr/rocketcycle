// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package postgresql

import (
	"context"
	"encoding/base64"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"

	"github.com/lachlanorr/rocketcycle/examples/rpg/pb"
)

type Player struct{}

func (*Player) Read(ctx context.Context, key string) (*pb.Player, *pb.PlayerRelated, *rkcy.CompoundOffset, error) {
	inst := &pb.Player{}
	var relB64 string
	cmpdOffset := &rkcy.CompoundOffset{}
	err := pool.QueryRow(
		ctx,
		`SELECT id,
                username,
                active,
                related,
                mro_generation,
                mro_partition,
                mro_offset
         FROM rpg.player
         WHERE id=$1`,
		key,
	).Scan(
		&inst.Id,
		&inst.Username,
		&inst.Active,
		&relB64,
		&cmpdOffset.Generation,
		&cmpdOffset.Partition,
		&cmpdOffset.Offset,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, nil, rkcy.NewError(rkcy.Code_NOT_FOUND, err.Error())
		}
		return nil, nil, nil, err
	}

	rel := &pb.PlayerRelated{}
	if relB64 != "" {
		relBytes, err := base64.StdEncoding.DecodeString(relB64)
		if err != nil {
			return nil, nil, nil, err
		}
		err = proto.Unmarshal(relBytes, rel)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return inst, rel, cmpdOffset, nil
}

func (p *Player) Create(ctx context.Context, inst *pb.Player, cmpdOffset *rkcy.CompoundOffset) (*pb.Player, error) {
	if inst.Id == "" {
		inst.Id = uuid.NewString()
	}
	err := p.upsert(ctx, inst, nil, cmpdOffset)
	if err != nil {
		return nil, err
	}
	return inst, nil
}

func (p *Player) Update(ctx context.Context, inst *pb.Player, rel *pb.PlayerRelated, cmpdOffset *rkcy.CompoundOffset) error {
	return p.upsert(ctx, inst, rel, cmpdOffset)
}

func (*Player) Delete(ctx context.Context, key string, cmpdOffset *rkcy.CompoundOffset) error {
	log.Warn().Msgf("Player DELETE %s", key)
	_, err := pool.Exec(
		context.Background(),
		"CALL rpg.sp_delete_player($1, $2, $3, $4)",
		key,
		cmpdOffset.Generation,
		cmpdOffset.Partition,
		cmpdOffset.Offset,
	)
	return err
}

func (*Player) upsert(ctx context.Context, inst *pb.Player, rel *pb.PlayerRelated, offset *rkcy.CompoundOffset) error {
	var relB64 string
	if rel != nil {
		relBytes, err := proto.Marshal(rel)
		if err != nil {
			return err
		}
		relB64 = base64.StdEncoding.EncodeToString(relBytes)
	}

	_, err := pool.Exec(
		ctx,
		"CALL rpg.sp_upsert_player($1, $2, $3, $4, $5, $6, $7)",
		inst.Id,
		inst.Username,
		inst.Active,
		relB64,
		offset.Generation,
		offset.Partition,
		offset.Offset,
	)
	return err
}
