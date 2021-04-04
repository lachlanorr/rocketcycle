// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package storage

import (
	"context"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

func (*Player) Read(ctx context.Context, args *rkcy.StepArgs) *rkcy.StepResult {
	rslt := rkcy.StepResult{}

	conn, err := connect(ctx)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_CONNECTION
		return &rslt
	}
	defer conn.Close(ctx)

	player := Player{}
	offset := rkcy.Offset{}
	err = conn.QueryRow(ctx, "SELECT id, username, active, mro_generation, mro_partition, mro_offset FROM rpg.player WHERE id=$1", args.Key).
		Scan(&player.Id, &player.Username, &player.Active, &offset.Generation, &offset.Partition, &offset.Offset)

	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_NOT_FOUND
		return &rslt
	}

	rslt.Offset = &offset

	rslt.Payload, err = Marshal(int32(ResourceType_PLAYER), &player)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}

	rslt.Code = rkcy.Code_OK
	return &rslt
}

func (*Player) upsert(ctx context.Context, args *rkcy.StepArgs) *rkcy.StepResult {
	rslt := rkcy.StepResult{}

	mdl, err := Unmarshal(args.Payload)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}
	plyr, ok := mdl.(*Player)
	if !ok {
		rslt.LogError("Unmarshal returned wrong type")
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}

	conn, err := connect(ctx)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_CONNECTION
		return &rslt
	}
	defer conn.Close(context.Background())

	_, err = conn.Exec(
		context.Background(),
		"CALL rpg.sp_upsert_player($1, $2, $3, $4, $5, $6)",
		args.Key,
		plyr.Username,
		plyr.Active,
		args.Offset.Generation,
		args.Offset.Partition,
		args.Offset.Offset,
	)

	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_INTERNAL
		return &rslt
	}

	rslt.Code = rkcy.Code_OK
	rslt.Payload = args.Payload
	return &rslt
}

func (p *Player) Create(ctx context.Context, args *rkcy.StepArgs) *rkcy.StepResult {
	return p.upsert(ctx, args)
}

func (p *Player) Update(ctx context.Context, args *rkcy.StepArgs) *rkcy.StepResult {
	return p.upsert(ctx, args)
}

func (*Player) Delete(ctx context.Context, args *rkcy.StepArgs) *rkcy.StepResult {
	rslt := rkcy.StepResult{}

	conn, err := connect(ctx)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_CONNECTION
		return &rslt
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(
		context.Background(),
		"CALL rpg.sp_delete_player($1, $2, $3, $4)",
		args.Key,
		args.Offset.Generation,
		args.Offset.Partition,
		args.Offset.Offset,
	)

	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_INTERNAL
		return &rslt
	}

	rslt.Code = rkcy.Code_OK
	return &rslt
}
