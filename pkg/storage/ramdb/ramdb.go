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
	items map[string]*Item
}

type Item struct {
	key        string
	inst       proto.Message
	rel        proto.Message
	cmpdOffset rkcypb.CompoundOffset
}

func NewRamDb() *RamDb {
	return &RamDb{
		items: make(map[string]*Item),
	}
}

func (ramdb *RamDb) Read(key string) (proto.Message, proto.Message, *rkcypb.CompoundOffset, error) {
	item, ok := ramdb.items[key]
	if !ok {
		return nil, nil, nil, fmt.Errorf("RamDb not found: %s", key)
	}
	cmpdOffset := item.cmpdOffset // copy to local so we don't return pointer to actual cmpdOffset in ramdb
	return proto.Clone(item.inst), proto.Clone(item.rel), &cmpdOffset, nil
}

func (ramdb *RamDb) Create(key string, inst proto.Message, cmpdOffset *rkcypb.CompoundOffset) (proto.Message, error) {
	return proto.Clone(inst), ramdb.Update(key, inst, nil, cmpdOffset)
}

func (ramdb *RamDb) Update(key string, inst proto.Message, rel proto.Message, cmpdOffset *rkcypb.CompoundOffset) error {
	offsetOk := true
	item, ok := ramdb.items[key]
	if ok {
		offsetOk = rkcy.OffsetGT(cmpdOffset, &item.cmpdOffset)
	}
	if offsetOk {
		ramdb.items[key] = &Item{
			key:        key,
			inst:       proto.Clone(inst),
			rel:        proto.Clone(rel),
			cmpdOffset: *cmpdOffset,
		}
	}
	return nil
}

func (ramdb *RamDb) Delete(key string, cmpdOffset *rkcypb.CompoundOffset) error {
	offsetOk := true
	item, ok := ramdb.items[key]
	if ok {
		offsetOk = rkcy.OffsetGT(cmpdOffset, &item.cmpdOffset)
	}
	if offsetOk {
		delete(ramdb.items, key)
	}
	return nil
}
