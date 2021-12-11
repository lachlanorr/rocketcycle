// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

func Contains(slice []string, item string) bool {
	for _, val := range slice {
		if val == item {
			return true
		}
	}
	return false
}

func Maxi(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func Mini(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func Maxi32(x, y int32) int32 {
	if x < y {
		return y
	}
	return x
}

func Mini32(x, y int32) int32 {
	if x > y {
		return y
	}
	return x
}

func Maxi64(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}

func Mini64(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}
