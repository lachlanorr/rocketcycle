// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"hash"
	"hash/fnv"

	"github.com/rs/zerolog/log"
)

var gHashFuncRegistry = make(map[string]HashFunc)

func init() {
	RegisterHashFunc("fnv64", NewFnv64HashFunc())
}

type HashFunc interface {
	Hash(val []byte, count int32) int32
}

type Fnv64HashFunc struct {
	fnv64 hash.Hash64
}

func NewFnv64HashFunc() *Fnv64HashFunc {
	return &Fnv64HashFunc{
		fnv64: fnv.New64(),
	}
}

func (fnv *Fnv64HashFunc) Hash(val []byte, count int32) int32 {
	fnv.fnv64.Reset()
	fnv.fnv64.Write(val)
	fnvCalc := fnv.fnv64.Sum64()
	partition := int32(fnvCalc % uint64(count))
	return partition
}

func GetHashFunc(name string) HashFunc {
	hashFunc, ok := gHashFuncRegistry[name]
	if !ok {
		log.Fatal().
			Msgf("HashFunc '%s' has not been registerd", name)
	}
	return hashFunc
}

func RegisterHashFunc(name string, hashFunc HashFunc) {
	if gHashFuncRegistry[name] != nil {
		log.Warn().
			Msgf("HashFunc '%s' already registered, replacing", name)
	}

	gHashFuncRegistry[name] = hashFunc
}
