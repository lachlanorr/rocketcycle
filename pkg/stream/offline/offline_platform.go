// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package offline

import (
	"context"
	"sync"

	"github.com/lachlanorr/rocketcycle/pkg/platform"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

func NewOfflinePlatformFromJson(
	ctx context.Context,
	platformDefJson []byte,
) (*platform.Platform, error) {
	rtPlatDef, err := rkcy.NewRtPlatformDefFromJson(platformDefJson)
	if err != nil {
		return nil, err
	}

	strmprov, err := NewOfflineStreamProvider(rtPlatDef)
	if err != nil {
		return nil, err
	}

	err = rkcy.CreatePlatformTopics(
		ctx,
		strmprov,
		rtPlatDef.PlatformDef.Name,
		rtPlatDef.PlatformDef.Environment,
		rtPlatDef.AdminCluster.Brokers,
	)
	if err != nil {
		return nil, err
	}

	err = rkcy.UpdateTopics(
		ctx,
		strmprov,
		rtPlatDef.PlatformDef,
	)
	if err != nil {
		return nil, err
	}

	var respTarget *rkcypb.TopicTarget
	if rtPlatDef.DefaultResponseTopic != nil {
		respTarget = &rkcypb.TopicTarget{
			Brokers:   rtPlatDef.DefaultResponseTopic.CurrentCluster.Brokers,
			Topic:     rtPlatDef.DefaultResponseTopic.CurrentTopic,
			Partition: 0,
		}
	}

	var wg sync.WaitGroup
	plat, err := platform.NewPlatform(
		ctx,
		&wg,
		rtPlatDef.PlatformDef.Name,
		rtPlatDef.PlatformDef.Environment,
		rtPlatDef.AdminCluster.Brokers,
		10,
		strmprov,
		respTarget,
	)
	if err != nil {
		return nil, err
	}

	return plat, nil
}
