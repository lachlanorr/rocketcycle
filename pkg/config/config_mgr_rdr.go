// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config

import (
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type ConfigMgrRdr struct {
	mgr *ConfigMgr
}

func NewConfigMgrRdr(mgr *ConfigMgr) *ConfigMgrRdr {
	return &ConfigMgrRdr{
		mgr: mgr,
	}
}

func (rdr *ConfigMgrRdr) GetString(key string) (string, bool) {
	return rdr.mgr.GetString(key)
}

func (rdr *ConfigMgrRdr) GetBool(key string) (bool, bool) {
	return rdr.mgr.GetBool(key)
}

func (rdr *ConfigMgrRdr) GetFloat64(key string) (float64, bool) {
	return rdr.mgr.GetFloat64(key)
}

func (rdr *ConfigMgrRdr) GetComplexMsg(msgType string, key string) (proto.Message, bool) {
	return rdr.mgr.GetComplexMsg(msgType, key)
}

func (rdr *ConfigMgrRdr) GetComplexBytes(msgType string, key string) ([]byte, bool) {
	return rdr.mgr.GetComplexBytes(msgType, key)
}

func (rdr *ConfigMgrRdr) BuildConfigResponse() *rkcypb.ConfigReadResponse {
	return rdr.mgr.BuildConfigResponse()
}
