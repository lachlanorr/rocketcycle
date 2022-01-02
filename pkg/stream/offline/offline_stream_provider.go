// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package offline

import (
	"context"
	"sync"
	"time"

	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/apecs"
	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type OfflineStreamProvider struct {
	mgr *Manager
}

func NewOfflineStreamProvider(rtPlatDef *rkcy.RtPlatformDef) *OfflineStreamProvider {
	return &OfflineStreamProvider{
		mgr: NewManager(rtPlatDef),
	}
}

func (ostrmprov *OfflineStreamProvider) NewConsumer(brokers string, groupName string, logCh chan kafka.LogEvent) (rkcy.Consumer, error) {
	clus, err := ostrmprov.GetCluster(brokers)
	if err != nil {
		return nil, err
	}

	return NewOfflineConsumer(clus), nil
}

func (ostrmprov *OfflineStreamProvider) NewProducer(brokers string, logCh chan kafka.LogEvent) (rkcy.Producer, error) {
	clus, err := ostrmprov.GetCluster(brokers)
	if err != nil {
		return nil, err
	}

	return NewOfflineProducer(clus), nil
}

func (ostrmprov *OfflineStreamProvider) NewAdminClient(brokers string) (rkcy.AdminClient, error) {
	return ostrmprov.GetCluster(brokers)
}
