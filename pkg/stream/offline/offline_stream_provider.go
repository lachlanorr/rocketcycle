// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package offline

import (
	"gopkg.in/confluentinc/confluent-kafka-go.v1/kafka"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

type OfflineStreamProvider struct {
	mgr *Manager
}

func NewOfflineStreamProvider(rtPlatDef *rkcy.RtPlatformDef) (*OfflineStreamProvider, error) {
	mgr, err := NewManager(rtPlatDef)
	if err != nil {
		return nil, err
	}
	return &OfflineStreamProvider{
		mgr: mgr,
	}, nil
}

func (ostrmprov *OfflineStreamProvider) NewConsumer(brokers string, groupName string, logCh chan kafka.LogEvent) (rkcy.Consumer, error) {
	clus, err := ostrmprov.mgr.GetCluster(brokers)
	if err != nil {
		return nil, err
	}

	return NewOfflineConsumer(clus), nil
}

func (ostrmprov *OfflineStreamProvider) NewProducer(brokers string, logCh chan kafka.LogEvent) (rkcy.Producer, error) {
	clus, err := ostrmprov.mgr.GetCluster(brokers)
	if err != nil {
		return nil, err
	}

	return NewOfflineProducer(clus), nil
}

func (ostrmprov *OfflineStreamProvider) NewAdminClient(brokers string) (rkcy.AdminClient, error) {
	return ostrmprov.mgr.GetCluster(brokers)
}
