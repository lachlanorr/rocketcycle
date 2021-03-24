// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

type InstanceCache struct {
	instances map[string][]byte
}

func NewInstanceCache() *InstanceCache {
	instCache := InstanceCache{
		instances: make(map[string][]byte),
	}
	return &instCache
}

func (instCache *InstanceCache) Get(key string) []byte {
	val, ok := instCache.instances[key]
	if ok {
		return val
	}
	return nil
}

func (instCache *InstanceCache) Set(key string, val []byte) {
	instCache.instances[key] = val
}
