// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"fmt"
	"strconv"
	"strings"
)

func DecimalFromString(s string) (Decimal, error) {
	if len(s) > 0 && s[0] == 'd' {
		parts := strings.Split(s[1:], ".")
		if len(parts) >= 1 && len(parts) <= 2 {
			var d Decimal
			var err error
			if len(parts[0]) > 0 { // handle cases like "d.1234" with no left side value
				d.L, err = strconv.ParseInt(parts[0], 10, 64)
			}
			if err == nil {
				if len(parts) == 1 || len(parts[1]) == 0 {
					// no fractional part, like "d1234."
					return d, nil
				}
				d.R, err = strconv.ParseInt(parts[1], 10, 64)
				if err == nil && d.R >= 0 { // no errors, and fractional part must be positive
					return d, nil
				}
			}
		}
	}
	return Decimal{}, fmt.Errorf("Badly formatted decimal string: %s", s)
}

func (d Decimal) ToString() string {
	return fmt.Sprintf("d%d.%d", d.L, d.R)
}
