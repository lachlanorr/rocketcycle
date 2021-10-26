// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package postgresql

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"

	"github.com/lachlanorr/rocketcycle/examples/rpg/consts"
	"github.com/lachlanorr/rocketcycle/examples/rpg/pb"
)

func init() {
	rkcy.RegisterCrudHandler("postgresql", "Character", &Character{})
}

type Character struct{}

func (c *Character) Read(ctx context.Context, key string) (*pb.Character, *pb.CharacterRelatedConcerns, *rkcy.CompoundOffset, error) {
	inst := &pb.Character{
		Currency: &pb.Character_Currency{},
	}
	cmpdOffset := rkcy.CompoundOffset{}
	err := pool().QueryRow(
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

	inst.Items, err = c.readItems(ctx, inst.Id)
	if err != nil {
		return nil, nil, nil, err
	}
	return inst, nil, &cmpdOffset, nil
}

func (c *Character) Create(ctx context.Context, inst *pb.Character, cmpdOffset *rkcy.CompoundOffset) (*pb.Character, error) {
	inst.Id = uuid.NewString()
	err := c.upsert(ctx, inst, nil, cmpdOffset)
	if err != nil {
		return nil, err
	}
	return inst, nil
}

func (c *Character) Update(ctx context.Context, inst *pb.Character, relCnc *pb.CharacterRelatedConcerns, cmpdOffset *rkcy.CompoundOffset) error {
	return c.upsert(ctx, inst, relCnc, cmpdOffset)
}

func (*Character) Delete(ctx context.Context, key string, cmpdOffset *rkcy.CompoundOffset) error {
	_, err := pool().Exec(
		ctx,
		"CALL rpg.sp_delete_character($1, $2, $3, $4)",
		key,
		cmpdOffset.Generation,
		cmpdOffset.Partition,
		cmpdOffset.Offset,
	)
	return err
}

func (*Character) readItems(ctx context.Context, key string) ([]*pb.Character_Item, error) {
	var items []*pb.Character_Item
	rows, err := pool().Query(ctx, "select id, description from rpg.character_item where character_id = $1", key)
	if err == nil {
		for rows.Next() {
			item := pb.Character_Item{}

			err = rows.Scan(&item.Id, &item.Description)
			if err != nil {
				return nil, err
			}
			items = append(items, &item)
		}
	}
	return items, nil
}

func (*Character) hasItem(id string, items []*pb.Character_Item) bool {
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

func (c *Character) upsert(ctx context.Context, inst *pb.Character, relCnc *pb.CharacterRelatedConcerns, cmpdOffset *rkcy.CompoundOffset) error {
	_, err := pool().Exec(
		ctx,
		"CALL rpg.sp_upsert_character($1, $2, $3, $4, $5, $6, $7)",
		inst.Id,
		inst.PlayerId,
		inst.Fullname,
		inst.Active,
		cmpdOffset.Generation,
		cmpdOffset.Partition,
		cmpdOffset.Offset,
	)
	if err != nil {
		return err
	}

	if inst.Currency == nil {
		inst.Currency = &pb.Character_Currency{}
	}
	_, err = pool().Exec(
		ctx,
		"CALL rpg.sp_upsert_character_currency($1, $2, $3, $4, $5, $6, $7, $8)",
		inst.Id,
		inst.Currency.Gold,
		inst.Currency.Faction_0,
		inst.Currency.Faction_1,
		inst.Currency.Faction_2,
		cmpdOffset.Generation,
		cmpdOffset.Partition,
		cmpdOffset.Offset,
	)
	if err != nil {
		return err
	}

	// retrieve all owned items so we can remove ones that may have
	// been removed
	dbItems, err := c.readItems(ctx, inst.Id)
	if err != nil {
		return err
	}
	for _, dbItem := range dbItems {
		if !c.hasItem(dbItem.Id, inst.Items) {
			_, err = pool().Exec(
				ctx,
				"CALL rpg.sp_upsert_character_item($1, $2, $3, $4, $5, $6)",
				dbItem.Id,
				consts.ZeroUuid,
				dbItem.Description,
				cmpdOffset.Generation,
				cmpdOffset.Partition,
				cmpdOffset.Offset,
			)

			if err != nil {
				return err
			}
		}
	}

	for _, item := range inst.Items {
		_, err = pool().Exec(
			ctx,
			"CALL rpg.sp_upsert_character_item($1, $2, $3, $4, $5, $6)",
			item.Id,
			inst.Id,
			item.Description,
			cmpdOffset.Generation,
			cmpdOffset.Partition,
			cmpdOffset.Offset,
		)

		if err != nil {
			return err
		}
	}

	return nil
}
