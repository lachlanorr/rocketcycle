// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package main

import (
	"github.com/lachlanorr/rkcy/examples/rpg/lib"
	"github.com/lachlanorr/rkcy/pkg/rkcy"
)

func main() {
	impl := rkcy.PlatformImpl{
		Name: lib.PlatformName,
	}

	rkcy.StartPlatform(&impl)
}
