// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package concerns

import (
	"context"

	"github.com/jackc/pgx/v4"

	"github.com/lachlanorr/rocketcycle/examples/rpg/consts"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

func (inst *Character) ReadItems(ctx context.Context, conn *pgx.Conn) ([]*Character_Item, error) {
	var items []*Character_Item
	rows, err := conn.Query(ctx, "select id, description from rpg.character_item where character_id = $1", inst.Id)
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

func (inst *Character) Read(ctx context.Context, key string) (*rkcy.Offset, error) {
	conn, err := connect(ctx)
	if err != nil {
		return nil, rkcy.NewError(rkcy.Code_CONNECTION, err.Error())
	}
	defer conn.Close(ctx)

	*inst = Character{
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
		key,
	).Scan(
		&inst.Id,
		&inst.PlayerId,
		&inst.Fullname,
		&inst.Active,
		&inst.Currency.Gold,
		&inst.Currency.Faction_0,
		&inst.Currency.Faction_1,
		&inst.Currency.Faction_2,
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

	inst.Items, err = inst.ReadItems(ctx, conn)
	if err != nil {
		return nil, err
	}
	return &offset, nil
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

func (inst *Character) upsert(ctx context.Context, offset *rkcy.Offset) error {
	conn, err := connect(ctx)
	if err != nil {
		return rkcy.NewError(rkcy.Code_CONNECTION, err.Error())
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(
		context.Background(),
		"CALL rpg.sp_upsert_character($1, $2, $3, $4, $5, $6, $7)",
		inst.Id,
		inst.PlayerId,
		inst.Fullname,
		inst.Active,
		offset.Generation,
		offset.Partition,
		offset.Offset,
	)
	if err != nil {
		return err
	}

	if inst.Currency == nil {
		inst.Currency = &Character_Currency{}
	}
	_, err = conn.Exec(
		context.Background(),
		"CALL rpg.sp_upsert_character_currency($1, $2, $3, $4, $5, $6, $7, $8)",
		inst.Id,
		inst.Currency.Gold,
		inst.Currency.Faction_0,
		inst.Currency.Faction_1,
		inst.Currency.Faction_2,
		offset.Generation,
		offset.Partition,
		offset.Offset,
	)
	if err != nil {
		return err
	}

	// retrieve all owned items so we can remove ones that may have
	// been removed
	dbItems, err := inst.ReadItems(ctx, conn)
	if err != nil {
		return err
	}
	for _, dbItem := range dbItems {
		if !hasItem(dbItem.Id, inst.Items) {
			_, err = conn.Exec(
				context.Background(),
				"CALL rpg.sp_upsert_character_item($1, $2, $3, $4, $5, $6)",
				dbItem.Id,
				consts.ZeroUuid,
				dbItem.Description,
				offset.Generation,
				offset.Partition,
				offset.Offset,
			)

			if err != nil {
				return err
			}
		}
	}
	for _, item := range inst.Items {
		_, err = conn.Exec(
			context.Background(),
			"CALL rpg.sp_upsert_character_item($1, $2, $3, $4, $5, $6)",
			item.Id,
			inst.Id,
			item.Description,
			offset.Generation,
			offset.Partition,
			offset.Offset,
		)

		if err != nil {
			return err
		}
	}

	return nil
}

func (inst *Character) Create(ctx context.Context, offset *rkcy.Offset) error {
	return inst.upsert(ctx, offset)
}

func (inst *Character) Update(ctx context.Context, offset *rkcy.Offset) error {
	return inst.upsert(ctx, offset)
}

func (inst *Character) Delete(ctx context.Context, key string, offset *rkcy.Offset) error {
	conn, err := connect(ctx)
	if err != nil {
		return rkcy.NewError(rkcy.Code_CONNECTION, err.Error())
	}
	defer conn.Close(ctx)

	_, err = conn.Exec(
		context.Background(),
		"CALL rpg.sp_delete_character($1, $2, $3, $4)",
		key,
		offset.Generation,
		offset.Partition,
		offset.Offset,
	)
	return err
}

func (*Character) ValidateCreate(ctx context.Context, payload *Character) (*Character, error) {
	if len(payload.Fullname) < 2 {
		return nil, rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Fullname too short")
	}
	return payload, nil
}

func (inst *Character) ValidateUpdate(ctx context.Context, payload *Character) (*Character, error) {
	return inst.ValidateCreate(ctx, payload)
}

func (inst *Character) Fund(ctx context.Context, payload *FundingRequest) (*Character, error) {
	if payload.Currency.Gold < 0 || payload.Currency.Faction_0 < 0 || payload.Currency.Faction_1 < 0 || payload.Currency.Faction_2 < 0 {
		return nil, rkcy.NewError(rkcy.Code_INVALID_ARGUMENT, "Cannot fund with negative currency")
	}

	if inst.Currency == nil {
		inst.Currency = &Character_Currency{}
	}

	inst.Currency.Gold += payload.Currency.Gold
	inst.Currency.Faction_0 += payload.Currency.Faction_0
	inst.Currency.Faction_1 += payload.Currency.Faction_1
	inst.Currency.Faction_2 += payload.Currency.Faction_2

	return inst, nil
}
