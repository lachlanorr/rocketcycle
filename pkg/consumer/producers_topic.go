// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package consumer

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type ProducerMessage struct {
	Directive         rkcypb.Directive
	ProducerDirective *rkcypb.ProducerDirective
	Timestamp         time.Time
	Offset            int64
}

func ConsumeProducersTopic(
	ctx context.Context,
	ch chan<- *ProducerMessage,
	adminBrokers string,
	platformName string,
	environment string,
	readyCh chan<- bool,
	wg *sync.WaitGroup,
) {
	wg.Add(1)
	go ConsumeMgmtTopic(
		ctx,
		adminBrokers,
		rkcy.ProducersTopic(platformName, environment),
		rkcypb.Directive_PRODUCER,
		PAST_LAST_MATCH,
		func(rawMsg *RawMessage) {
			prodDir := &rkcypb.ProducerDirective{}
			err := proto.Unmarshal(rawMsg.Value, prodDir)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to Unmarshal ProducerDirective")
				return
			}

			ch <- &ProducerMessage{
				Directive:         rawMsg.Directive,
				Timestamp:         rawMsg.Timestamp,
				Offset:            rawMsg.Offset,
				ProducerDirective: prodDir,
			}
		},
		readyCh,
		wg,
	)
}
