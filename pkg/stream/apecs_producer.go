// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package stream

import (
	"context"
	"fmt"
	"sync"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

type ManagedProducerMap map[rkcy.StandardTopicName]*ManagedProducer

type ApecsProducer struct {
	platArgs *rkcy.PlatformArgs
	bprod    *BrokersProducer

	producers    map[string]ManagedProducerMap
	producersMtx sync.Mutex
}

func NewApecsProducer(
	ctx context.Context,
	wg *sync.WaitGroup,
	platArgs *rkcy.PlatformArgs,
	bprod *BrokersProducer,
) *ApecsProducer {
	aprod := &ApecsProducer{
		platArgs: platArgs,
		bprod:    bprod,

		producers: make(map[string]ManagedProducerMap),
	}

	return aprod
}

func (aprod *ApecsProducer) GetManagedProducer(
	ctx context.Context,
	wg *sync.WaitGroup,
	concernName string,
	topicName rkcy.StandardTopicName,
) (*ManagedProducer, error) {
	aprod.producersMtx.Lock()
	defer aprod.producersMtx.Unlock()

	concernProds, ok := aprod.producers[concernName]
	if !ok {
		concernProds = make(ManagedProducerMap)
		aprod.producers[concernName] = concernProds
	}
	mprod, ok := concernProds[topicName]
	if !ok {
		mprod = NewManagedProducer(
			ctx,
			wg,
			aprod.platArgs,
			aprod.bprod,
			concernName,
			string(topicName),
		)

		if mprod == nil {
			return nil, fmt.Errorf(
				"ApecsProducer.GetManagedProducer Brokers=%s Platform=%s Concern=%s Topic=%s: Failed to create Producer",
				aprod.platArgs.AdminBrokers,
				aprod.platArgs.Platform,
				concernName,
				topicName,
			)
		}
		concernProds[topicName] = mprod
	}
	return mprod, nil
}
