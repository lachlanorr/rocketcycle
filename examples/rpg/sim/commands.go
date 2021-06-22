// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package sim

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/rs/zerolog"

	"github.com/lachlanorr/rocketcycle/examples/rpg/concerns"
	"github.com/lachlanorr/rocketcycle/examples/rpg/edge"
)

func cmdCreateCharacters(ctx context.Context, client edge.RpgServiceClient, r *rand.Rand, count uint) {
	slog, ok := ctx.Value("logger").(zerolog.Logger)
	if !ok {
		fmt.Printf("no logger provided")
		return
	}

	for i := uint(0); i < count; i++ {
		player, err := client.CreatePlayer(
			context.Background(),
			&concerns.Player{
				Username: fmt.Sprintf("User_%d", r.Int31()),
				Active:   true,
			})
		if err != nil {
			slog.Error().
				Err(err).
				Msg("Failed to CreatePlayer")
			return
		}
		slog.Info().Msgf("Player created %s", player.Id)
	}
}
