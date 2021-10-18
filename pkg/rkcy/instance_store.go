// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"time"

	"github.com/rs/zerolog/log"
)

type InstanceRecord struct {
	Instance   []byte
	Related    []byte
	CmpdOffset *CompoundOffset
	LastAccess time.Time
}

type InstanceStore struct {
	instances map[string]*InstanceRecord
}

func NewInstanceStore() *InstanceStore {
	instStore := InstanceStore{
		instances: make(map[string]*InstanceRecord),
	}
	return &instStore
}

func (instStore *InstanceStore) Get(key string) *InstanceRecord {
	rec, ok := instStore.instances[key]
	if ok {
		rec.LastAccess = time.Now()
		return rec
	}
	return nil
}

func (instStore *InstanceStore) GetInstance(key string) []byte {
	rec := instStore.Get(key)
	return rec.Instance
}

func (instStore *InstanceStore) GetRelated(key string) []byte {
	rec := instStore.Get(key)
	return rec.Related
}

func (instStore *InstanceStore) SetInstance(key string, instance []byte, cmpdOffset *CompoundOffset) {
	rec, ok := instStore.instances[key]
	if ok {
		if !OffsetGreaterThan(cmpdOffset, rec.CmpdOffset) {
			log.Error().
				Str("Key", key).
				Msgf("Out of order InstanceStore.Set: new (%+v), old (%+v)", cmpdOffset, rec.CmpdOffset)
			return
		}
		rec.Instance = instance
		rec.CmpdOffset = cmpdOffset
		rec.LastAccess = time.Now()
	} else {
		instStore.instances[key] = &InstanceRecord{Instance: instance, CmpdOffset: cmpdOffset, LastAccess: time.Now()}
	}
}

func (instStore *InstanceStore) SetRelated(key string, related []byte, cmpdOffset *CompoundOffset) {
	rec, ok := instStore.instances[key]
	if ok {
		if !OffsetGreaterThan(cmpdOffset, rec.CmpdOffset) {
			log.Error().
				Str("Key", key).
				Msgf("Out of order InstanceStore.Set: new (%+v), old (%+v)", cmpdOffset, rec.CmpdOffset)
			return
		}
		rec.Related = related
		rec.CmpdOffset = cmpdOffset
		rec.LastAccess = time.Now()
	}
	log.Error().
		Str("Key", key).
		Msgf("Unable to set related on missing key")
}

func (instStore *InstanceStore) Remove(key string) {
	delete(instStore.instances, key)
}
