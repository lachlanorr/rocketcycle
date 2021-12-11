// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package consumer

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

func ConsumePlatformTopic(
	ctx context.Context,
	plat rkcy.Platform,
	ch chan<- *rkcy.PlatformMessage,
	readyCh chan<- bool,
	wg *sync.WaitGroup,
) {
	var currRtPlatDef *rkcy.RtPlatformDef

	wg.Add(1)
	go ConsumeMgmtTopic(
		ctx,
		plat,
		rkcy.PlatformTopic(plat.Name(), plat.Environment()),
		rkcypb.Directive_PLATFORM,
		AT_LAST_MATCH,
		func(rawMsg *RawMessage) {
			platDef := &rkcypb.PlatformDef{}
			err := proto.Unmarshal(rawMsg.Value, platDef)
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to Unmarshal Platform")
				return
			}

			rtPlatDef, err := rkcy.NewRtPlatformDef(platDef, plat.Name(), plat.Environment())
			if err != nil {
				log.Error().
					Err(err).
					Msg("Failed to newRtPlatform")
				return
			}

			if currRtPlatDef != nil {
				if rtPlatDef.Hash == currRtPlatDef.Hash {
					// this happens frequently when admin replublishes
					return
				}
				if !rtPlatDef.PlatformDef.UpdateTime.AsTime().After(currRtPlatDef.PlatformDef.UpdateTime.AsTime()) {
					log.Info().
						Msgf("Platform not newer: old(%s) vs new(%s)", currRtPlatDef.PlatformDef.UpdateTime.AsTime(), rtPlatDef.PlatformDef.UpdateTime.AsTime())
					return
				}
			}

			ch <- &rkcy.PlatformMessage{
				Directive:    rawMsg.Directive,
				Timestamp:    rawMsg.Timestamp,
				Offset:       rawMsg.Offset,
				NewRtPlatDef: rtPlatDef,
				OldRtPlatDef: currRtPlatDef,
			}
		},
		readyCh,
		wg,
	)
}
