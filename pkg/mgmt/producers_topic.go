// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package mgmt

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
	wg *sync.WaitGroup,
	strmprov rkcy.StreamProvider,
	platform string,
	environment string,
	adminBrokers string,
	ch chan<- *ProducerMessage,
	readyCh chan<- bool,
) {
	wg.Add(1)
	go ConsumeMgmtTopic(
		ctx,
		wg,
		strmprov,
		adminBrokers,
		rkcy.ProducersTopic(platform, environment),
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
	)
}
