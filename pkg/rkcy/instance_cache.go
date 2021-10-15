// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"github.com/rs/zerolog/log"
)

type cachedBuffer struct {
	buffer     []byte
	cmpdOffset *CompoundOffset
}

type InstanceCache struct {
	instances map[string]*cachedBuffer
}

func NewInstanceCache() *InstanceCache {
	instCache := InstanceCache{
		instances: make(map[string]*cachedBuffer),
	}
	return &instCache
}

func (instCache *InstanceCache) Get(key string) []byte {
	val, ok := instCache.instances[key]
	if ok {
		return val.buffer
	}
	return nil
}

func (instCache *InstanceCache) Set(key string, val []byte, cmpdOffset *CompoundOffset) {
	oldVal, ok := instCache.instances[key]
	if ok {
		if !OffsetGreaterThan(cmpdOffset, oldVal.cmpdOffset) {
			log.Error().
				Str("Key", key).
				Msgf("Out of order InstanceCache.Set: new (%+v), old (%+v)", cmpdOffset, oldVal.cmpdOffset)
			return
		}
	}
	instCache.instances[key] = &cachedBuffer{buffer: val, cmpdOffset: cmpdOffset}
}

func (instCache *InstanceCache) Remove(key string) {
	delete(instCache.instances, key)
}
