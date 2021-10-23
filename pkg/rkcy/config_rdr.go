// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"google.golang.org/protobuf/proto"
)

type ConfigRdr struct {
	mgr *ConfigMgr
}

func NewConfigRdr(mgr *ConfigMgr) *ConfigRdr {
	return &ConfigRdr{
		mgr: mgr,
	}
}

func (rdr *ConfigRdr) GetString(key string) (string, bool) {
	return rdr.mgr.GetString(key)
}

func (rdr *ConfigRdr) GetBool(key string) (bool, bool) {
	return rdr.mgr.GetBool(key)
}

func (rdr *ConfigRdr) GetFloat64(key string) (float64, bool) {
	return rdr.mgr.GetFloat64(key)
}

func (rdr *ConfigRdr) GetComplexMsg(msgType string, key string) (proto.Message, bool) {
	return rdr.mgr.GetComplexMsg(msgType, key)
}

func (rdr *ConfigRdr) GetComplexBytes(msgType string, key string) ([]byte, bool) {
	return rdr.mgr.GetComplexBytes(msgType, key)
}
