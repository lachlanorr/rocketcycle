// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package storage

import (
	"context"

	"github.com/jackc/pgx/v4"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/examples/rpg/consts"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

func (*Character) ReadItems(ctx context.Context, conn *pgx.Conn, characterId string) ([]*Character_Item, error) {
	var items []*Character_Item
	rows, err := conn.Query(ctx, "select id, description from rpg.character_item where character_id = $1", characterId)
	if err == nil {
		for rows.Next() {
			item := Character_Item{}

			err = rows.Scan(&item.Id, &item.Description)
			if err != nil {
				return nil, err
			}
			items = append(items, &item)
		}
	}
	return items, nil
}

func (c *Character) Read(ctx context.Context, args *rkcy.StepArgs) *rkcy.StepResult {
	rslt := rkcy.StepResult{}

	conn, err := connect(ctx)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_CONNECTION
		return &rslt
	}
	defer conn.Close(ctx)

	character := Character{
		Currency: &Character_Currency{},
	}

	offset := rkcy.Offset{}
	err = conn.QueryRow(
		ctx,
		`SELECT c.id,
                c.player_id,
                c.fullname,
                c.active,
                cc.gold,
                cc.faction_0,
                cc.faction_1,
                cc.faction_2,
                c.mro_generation,
                c.mro_partition,
                c.mro_offset
           FROM rpg.character c
           JOIN rpg.character_currency cc on (c.id = cc.character_id)
          WHERE id=$1`,
		args.Key,
	).Scan(
		&character.Id,
		&character.PlayerId,
		&character.Fullname,
		&character.Active,
		&character.Currency.Gold,
		&character.Currency.Faction_0,
		&character.Currency.Faction_1,
		&character.Currency.Faction_2,
		&offset.Generation,
		&offset.Partition,
		&offset.Offset,
	)

	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_NOT_FOUND
		return &rslt
	}

	rslt.Offset = &offset

	character.Items, err = c.ReadItems(ctx, conn, args.Key)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_INTERNAL
		return &rslt
	}

	rslt.Payload, err = proto.Marshal(&character)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_MARSHAL_FAILED
		return &rslt
	}

	rslt.Code = rkcy.Code_OK
	return &rslt
}

func hasItem(id string, items []*Character_Item) bool {
	if items == nil {
		return false
	}

	for _, item := range items {
		if item.Id == id {
			return true
		}
	}
	return false
}

func (c *Character) upsert(ctx context.Context, args *rkcy.StepArgs) *rkcy.StepResult {
	rslt := rkcy.StepResult{}

	mdl := Character{}
	err := proto.Unmarshal(args.Payload, &mdl)
	if err != nil {
		rslt.LogError(err.Error())
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
		"CALL rpg.sp_upsert_character($1, $2, $3, $4, $5, $6, $7)",
		args.Key,
		mdl.PlayerId,
		mdl.Fullname,
		mdl.Active,
		args.Offset.Generation,
		args.Offset.Partition,
		args.Offset.Offset,
	)

	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_INTERNAL
		return &rslt
	}

	_, err = conn.Exec(
		context.Background(),
		"CALL rpg.sp_upsert_character_currency($1, $2, $3, $4, $5, $6, $7, $8)",
		args.Key,
		mdl.Currency.Gold,
		mdl.Currency.Faction_0,
		mdl.Currency.Faction_1,
		mdl.Currency.Faction_2,
		args.Offset.Generation,
		args.Offset.Partition,
		args.Offset.Offset,
	)

	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_INTERNAL
		return &rslt
	}

	// retrieve all owned items so we can remove ones that may have
	// been removed
	dbItems, err := c.ReadItems(ctx, conn, args.Key)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy.Code_INTERNAL
		return &rslt
	}
	for _, dbItem := range dbItems {
		if !hasItem(dbItem.Id, mdl.Items) {
			_, err = conn.Exec(
				context.Background(),
				"CALL rpg.sp_upsert_character_item($1, $2, $3, $4, $5, $6)",
				dbItem.Id,
				consts.ZeroUuid,
				dbItem.Description,
				args.Offset.Generation,
				args.Offset.Partition,
				args.Offset.Offset,
			)

			if err != nil {
				rslt.LogError(err.Error())
				rslt.Code = rkcy.Code_INTERNAL
				return &rslt
			}
		}
	}
	for _, item := range mdl.Items {
		_, err = conn.Exec(
			context.Background(),
			"CALL rpg.sp_upsert_character_item($1, $2, $3, $4, $5, $6)",
			item.Id,
			args.Key,
			item.Description,
			args.Offset.Generation,
			args.Offset.Partition,
			args.Offset.Offset,
		)

		if err != nil {
			rslt.LogError(err.Error())
			rslt.Code = rkcy.Code_INTERNAL
			return &rslt
		}
	}

	rslt.Code = rkcy.Code_OK
	rslt.Payload = args.Payload
	return &rslt
}

func (p *Character) Create(ctx context.Context, args *rkcy.StepArgs) *rkcy.StepResult {
	return p.upsert(ctx, args)
}

func (p *Character) Update(ctx context.Context, args *rkcy.StepArgs) *rkcy.StepResult {
	return p.upsert(ctx, args)
}

func (*Character) Delete(ctx context.Context, args *rkcy.StepArgs) *rkcy.StepResult {
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
		"CALL rpg.sp_delete_character($1, $2, $3, $4)",
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
