// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package ramdb

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type RamDb struct {
	concerns map[string]ItemMap
}

type ItemMap map[string]*Item

type Item struct {
	key        string
	inst       proto.Message
	rel        proto.Message
	cmpdOffset rkcypb.CompoundOffset
}

func NewRamDb() *RamDb {
	return &RamDb{
		concerns: make(map[string]ItemMap),
	}
}

func (ramdb *RamDb) Read(concern string, key string) (proto.Message, proto.Message, *rkcypb.CompoundOffset, error) {
	itemMap, ok := ramdb.concerns[concern]
	if !ok {
		return nil, nil, nil, fmt.Errorf("RamDb not found: %s", key)
	}

	item, ok := itemMap[key]
	if !ok {
		return nil, nil, nil, fmt.Errorf("RamDb not found: %s", key)
	}
	cmpdOffset := item.cmpdOffset // copy to local so we don't return pointer to actual cmpdOffset in ramdb
	return proto.Clone(item.inst), proto.Clone(item.rel), &cmpdOffset, nil
}

func (ramdb *RamDb) Create(concern string, key string, inst proto.Message, cmpdOffset *rkcypb.CompoundOffset) error {
	return ramdb.Update(concern, key, inst, nil, cmpdOffset)
}

func (ramdb *RamDb) Update(concern string, key string, inst proto.Message, rel proto.Message, cmpdOffset *rkcypb.CompoundOffset) error {
	itemMap, ok := ramdb.concerns[concern]
	if !ok {
		itemMap = make(map[string]*Item)
		ramdb.concerns[concern] = itemMap
	}

	offsetOk := true
	item, ok := itemMap[key]
	if ok {
		offsetOk = rkcy.OffsetGT(cmpdOffset, &item.cmpdOffset)
	}
	if offsetOk {
		itemMap[key] = &Item{
			key:        key,
			inst:       proto.Clone(inst),
			rel:        proto.Clone(rel),
			cmpdOffset: *cmpdOffset,
		}
	}
	return nil
}

func (ramdb *RamDb) Delete(concern string, key string, cmpdOffset *rkcypb.CompoundOffset) error {
	itemMap, ok := ramdb.concerns[concern]
	if !ok {
		return nil
	}

	offsetOk := true
	item, ok := itemMap[key]
	if ok {
		offsetOk = rkcy.OffsetGT(cmpdOffset, &item.cmpdOffset)
	}
	if offsetOk {
		delete(itemMap, key)
	}
	return nil
}
