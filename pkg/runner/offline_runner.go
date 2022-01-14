// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package runner

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/lachlanorr/rocketcycle/pkg/apecs"
	"github.com/lachlanorr/rocketcycle/pkg/platform"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
	"github.com/lachlanorr/rocketcycle/pkg/stream/offline"
)

type OfflineRunner struct {
	rkcycmd *RkcyCmd
	plat    *platform.Platform
}

func NewOfflineRunner(
	platformDefJson []byte,
	configJson []byte,
	clientCode *rkcy.ClientCode,
) (*OfflineRunner, error) {
	ornr := &OfflineRunner{}

	rtPlatDef, err := rkcy.NewRtPlatformDefFromJson(platformDefJson)
	if err != nil {
		return nil, err
	}

	if rtPlatDef.DefaultResponseTopic == nil {
		return nil, errors.New("No DefaultResponseTopic available in PlatformDef")
	}

	readyCh := make(chan bool)

	ornr.rkcycmd = newRkcyCmd(
		context.Background(),
		rtPlatDef.PlatformDef.Name,
		clientCode,
		nil,
		nil,
		readyCh,
	)
	ornr.rkcycmd.setPlatformDefJson(platformDefJson)
	ornr.rkcycmd.setConfigJson(configJson)

	cobraCmd := ornr.rkcycmd.BuildCobraCommand()
	cobraCmd.SetArgs([]string{"run", "-d", "-e", rtPlatDef.PlatformDef.Environment, "--runner", "routine", "--admin_brokers", rtPlatDef.AdminCluster.Brokers})
	go cobraCmd.ExecuteContext(ornr.rkcycmd.ctx)

	_ = <-readyCh

	respTarget := &rkcypb.TopicTarget{
		Brokers:   rtPlatDef.DefaultResponseTopic.CurrentCluster.Brokers,
		Topic:     rtPlatDef.DefaultResponseTopic.CurrentTopic,
		Partition: 0,
	}

	ornr.plat, err = platform.NewPlatform(
		ornr.rkcycmd.ctx,
		&ornr.rkcycmd.wg,
		uuid.NewString(),
		ornr.rkcycmd.platform,
		ornr.rkcycmd.settings.Environment,
		ornr.rkcycmd.settings.AdminBrokers,
		time.Duration(ornr.rkcycmd.settings.AdminPingIntervalSecs)*time.Second,
		offline.NewOfflineStreamProviderFromManager(ornr.rkcycmd.offlineMgr),
		ornr.rkcycmd.clientCode,
		respTarget,
	)
	if err != nil {
		ornr.Close()
		return nil, err
	}

	return ornr, nil
}

func (ornr *OfflineRunner) Close() {
	ornr.rkcycmd.ctxCancel()
	ornr.rkcycmd.wg.Wait()
}

func (ornr *OfflineRunner) Platform() *platform.Platform {
	return ornr.plat
}

func (ornr *OfflineRunner) OfflineManager() *offline.Manager {
	return ornr.rkcycmd.offlineMgr
}

func (ornr *OfflineRunner) ExecuteTxnSync(txn *rkcy.Txn) (*rkcy.ResultProto, error) {
	return apecs.ExecuteTxnSync(
		ornr.plat.Context(),
		ornr.plat,
		txn,
		time.Second*10,
	)
}
