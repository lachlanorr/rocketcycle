// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package program

import (
	"fmt"
)

// reasonable list of colors that change greatly each time
var gColors []int = []int{
	11, 12, /*13, 14, 10,*/ /*9,*/
	31 /*47,*/ /*63,*/, 79, 95 /*111,*/, 127, 143, 159, 175 /*191,*/, 207, 223,
	25, 41, 57, 73, 89, 105, 121, 137, 153, 169, 185, 201, 217,
	26, 42, 58, 74, 90, 106, 122, 138, 154, 170, 186, 202, 218,
	27, 43, 59, 75, 91, 107, 123, 139, 155, 171, 187, 203, 219,
	28, 44, 60, 76, 92, 108, 124, 140, 156, 172, 188, 204, 220,
	29, 45, 61, 77, 93, 109, 125, 141, 157, 173, 189, 205, 221,
}

func colorize(s interface{}, c int) string {
	return fmt.Sprintf("\x1b[38;5;%dm%v\x1b[0m", c, s)
}
