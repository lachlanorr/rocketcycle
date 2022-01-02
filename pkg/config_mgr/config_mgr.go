// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package config_mgr

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/lachlanorr/rocketcycle/pkg/mgmt"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type ConfigMgr struct {
	config            *rkcy.Config
	lastChanged       time.Time
	lastChangedOffset int64

	mtx sync.Mutex
}

func NewConfigMgr(
	ctx context.Context,
	wg *sync.WaitGroup,
	strmprov rkcy.StreamProvider,
	platform string,
	environment string,
	adminBrokers string,
) *ConfigMgr {
	confMgr := &ConfigMgr{
		config: rkcy.NewConfig(),
	}

	wg.Add(1)
	go confMgr.manageConfigTopic(
		ctx,
		wg,
		strmprov,
		platform,
		environment,
		adminBrokers,
	)

	return confMgr
}

func (confMgr *ConfigMgr) GetString(key string) (string, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil {
		return confMgr.config.GetString(key)
	}
	return "", false
}

func (confMgr *ConfigMgr) SetString(key string, val string) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = rkcy.NewConfig()
	}
	confMgr.config.SetString(key, val)
}

func (confMgr *ConfigMgr) GetBool(key string) (bool, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil {
		return confMgr.config.GetBool(key)
	}
	return false, false
}

func (confMgr *ConfigMgr) SetBool(key string, val bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = rkcy.NewConfig()
	}
	confMgr.config.SetBool(key, val)
}

func (confMgr *ConfigMgr) GetFloat64(key string) (float64, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil {
		return confMgr.config.GetFloat64(key)
	}
	return 0.0, false
}

func (confMgr *ConfigMgr) SetFloat64(key string, val float64) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = rkcy.NewConfig()
	}
	confMgr.config.SetFloat64(key, val)
}

func (confMgr *ConfigMgr) GetComplexMsg(msgType string, key string) (proto.Message, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil {
		return confMgr.config.GetComplexMsg(msgType, key)
	}
	return nil, false
}

func (confMgr *ConfigMgr) SetComplexMsg(msgType string, key string, msg proto.Message) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = rkcy.NewConfig()
	}
	confMgr.config.SetComplexMsg(msgType, key, msg)
}

func (confMgr *ConfigMgr) GetComplexBytes(msgType string, key string) ([]byte, bool) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config != nil {
		return confMgr.config.GetComplexBytes(msgType, key)
	}
	return nil, false
}

func (confMgr *ConfigMgr) SetComplexBytes(msgType string, key string, val []byte) {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()
	if confMgr.config == nil {
		confMgr.config = rkcy.NewConfig()
	}
	confMgr.config.SetComplexBytes(msgType, key, val)
}

func (confMgr *ConfigMgr) manageConfigTopic(
	ctx context.Context,
	wg *sync.WaitGroup,
	strmprov rkcy.StreamProvider,
	platform string,
	environment string,
	adminBrokers string,
) {
	defer wg.Done()

	confPubCh := make(chan *mgmt.ConfigPublishMessage)

	mgmt.ConsumeConfigTopic(
		ctx,
		wg,
		strmprov,
		platform,
		environment,
		adminBrokers,
		confPubCh,
		nil,
	)

	for {
		select {
		case <-ctx.Done():
			return
		case confPubMsg := <-confPubCh:
			confMgr.mtx.Lock()
			confMgr.config = &rkcy.Config{Config: *confPubMsg.Config}
			confMgr.lastChanged = confPubMsg.Timestamp
			confMgr.lastChangedOffset = confPubMsg.Offset
			confMgr.mtx.Unlock()
		}
	}
}

func (confMgr *ConfigMgr) BuildConfigResponse() *rkcypb.ConfigReadResponse {
	confMgr.mtx.Lock()
	defer confMgr.mtx.Unlock()

	confRsp := &rkcypb.ConfigReadResponse{
		Config:            &confMgr.config.Config,
		LastChanged:       timestamppb.New(confMgr.lastChanged),
		LastChangedOffset: confMgr.lastChangedOffset,
	}

	var err error
	confRsp.ConfigJson, err = rkcy.ConfigToJson(confMgr.config)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to convert config to json")
		confRsp.ConfigJson = "{\"error\": \"error converting config to json\"}"
	}

	return confRsp
}
