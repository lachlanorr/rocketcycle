// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package offline

import (
	"context"
	"sync"
	"time"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type OfflineApecsProducer struct {
}

func NewOfflineApecsProducer(
	ctx context.Context,
	plat rkcy.Platform,
	respTarget *rkcypb.TopicTarget,
	wg *sync.WaitGroup,
) *OfflineApecsProducer {
	return &OfflineApecsProducer{}
}

func (oaprod *OfflineApecsProducer) ResponseTarget() *rkcypb.TopicTarget {
	return nil
}

func (oaprod *OfflineApecsProducer) ResponseChannel(
	ctx context.Context,
	brokers string,
	wg *sync.WaitGroup,
) rkcy.ProducerCh {
	return nil
}

func (oaprod *OfflineApecsProducer) GetManagedProducer(
	concernName string,
	topicName rkcy.StandardTopicName,
	wg *sync.WaitGroup,
) (rkcy.ManagedProducer, error) {
	return nil, nil
}

func (oaprod *OfflineApecsProducer) ExecuteTxnSync(
	ctx context.Context,
	txn *rkcy.Txn,
	timeout time.Duration,
	wg *sync.WaitGroup,
) (*rkcy.ResultProto, error) {
	return nil, nil
}
