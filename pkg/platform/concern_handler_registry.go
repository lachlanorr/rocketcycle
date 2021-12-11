// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package platform

import (
	"fmt"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

var gConcernHandlerRegistry = make(rkcy.ConcernHandlers)

func RegisterGlobalConcernHandler(cncHandler rkcy.ConcernHandler) {
	_, ok := gConcernHandlerRegistry[cncHandler.ConcernName()]
	if ok {
		panic(fmt.Sprintf("%s concern handler already registered", cncHandler.ConcernName()))
	}
	gConcernHandlerRegistry[cncHandler.ConcernName()] = cncHandler
}
