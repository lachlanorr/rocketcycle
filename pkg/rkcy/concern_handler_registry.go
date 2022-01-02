// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"fmt"
)

var gConcernHandlerRegistry = make(ConcernHandlers)

func RegisterGlobalConcernHandler(cncHandler ConcernHandler) {
	_, ok := gConcernHandlerRegistry[cncHandler.ConcernName()]
	if ok {
		panic(fmt.Sprintf("%s concern handler already registered", cncHandler.ConcernName()))
	}
	gConcernHandlerRegistry[cncHandler.ConcernName()] = cncHandler
}

func GlobalConcernHandlerRegistry() ConcernHandlers {
	return gConcernHandlerRegistry
}
