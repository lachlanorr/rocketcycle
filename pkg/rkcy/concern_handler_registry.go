// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"fmt"
	"sync"
)

var gConcernHandlerNewFuncRegistry = make([]func() ConcernHandler, 0)
var gConcernHandlerNewFuncRegistryMtx sync.Mutex

func RegisterGlobalConcernHandlerNewFunc(newCncHdlr func() ConcernHandler) {
	gConcernHandlerNewFuncRegistryMtx.Lock()
	defer gConcernHandlerNewFuncRegistryMtx.Unlock()
	gConcernHandlerNewFuncRegistry = append(gConcernHandlerNewFuncRegistry, newCncHdlr)
}

func NewGlobalConcernHandlerRegistry() ConcernHandlers {
	gConcernHandlerNewFuncRegistryMtx.Lock()
	defer gConcernHandlerNewFuncRegistryMtx.Unlock()

	cncHdlrs := NewConcernHandlers()

	for _, newCncHdlr := range gConcernHandlerNewFuncRegistry {
		cncHdlr := newCncHdlr()
		_, ok := cncHdlrs[cncHdlr.ConcernName()]
		if ok {
			panic(fmt.Sprintf("%s concern handler registered multiple times", cncHdlr.ConcernName()))
		}
		cncHdlrs[cncHdlr.ConcernName()] = cncHdlr
	}

	return cncHdlrs
}
