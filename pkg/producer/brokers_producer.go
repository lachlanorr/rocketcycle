// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package producer

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
)

type BrokersProducer struct {
	cprods    map[string]*ChanneledProducer
	cprodsMtx *sync.Mutex
	telem     *rkcy.Telem
}

func NewBrokersProducer(telem *rkcy.Telem) *BrokersProducer {
	return &BrokersProducer{
		cprods:    make(map[string]*ChanneledProducer),
		cprodsMtx: &sync.Mutex{},
		telem:     telem,
	}
}

func (bprod *BrokersProducer) GetProducerCh(ctx context.Context, brokers string, wg *sync.WaitGroup) rkcy.ProducerCh {
	bprod.cprodsMtx.Lock()
	defer bprod.cprodsMtx.Unlock()

	cp, ok := bprod.cprods[brokers]
	if ok {
		return cp.ch
	}

	var err error
	cp, err = NewChanneledProducer(ctx, brokers, bprod.telem)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("Brokers", brokers).
			Msgf("Failed to NewChanneledProducer")
		return nil
	}
	bprod.cprods[brokers] = cp

	wg.Add(1)
	go cp.Run(ctx, wg)
	return cp.ch
}
