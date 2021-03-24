// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package storage

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcy"

	rpg_pb "github.com/lachlanorr/rocketcycle/examples/rpg/pb"
	rkcy_pb "github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
)

func connect() (*pgx.Conn, error) {
	// LORRTODO: make connection string a config
	return pgx.Connect(context.Background(), "postgresql://postgres@127.0.0.1:5432/rpg")
}

type Player struct {
}

func (player *Player) Read(ctx context.Context, args *rkcy.StepArgs) *rkcy.StepResult {
	log.Info().
		Msg("storage/player.go/Read")
	return nil
}

func (player *Player) Create(ctx context.Context, args *rkcy.StepArgs) *rkcy.StepResult {
	log.Info().
		Msg("storage/player.go/Create")

	rslt := rkcy.StepResult{}

	mdl := rpg_pb.Player{}
	err := proto.Unmarshal(args.Payload, &mdl)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy_pb.Code_MARSHAL_FAILED
		return &rslt
	}

	conn, err := connect()
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy_pb.Code_CONNECTION
		return &rslt
	}
	defer conn.Close(context.Background())

	stmt := `INSERT INTO rpg.player (id, username, active, mro_generation, mro_partition, mro_offset)
             VALUES ($1, $2, $3, $4, $5, $6)
             RETURNING id`

	retid := ""
	err = conn.QueryRow(
		context.Background(),
		stmt,
		mdl.Id,
		mdl.Username,
		mdl.Active,
		args.Offset.Generation,
		args.Offset.Partition,
		args.Offset.Offset,
	).Scan(&retid)

	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy_pb.Code_INTERNAL
		return &rslt
	}

	rslt.Payload, err = proto.Marshal(&mdl)
	if err != nil {
		rslt.LogError(err.Error())
		rslt.Code = rkcy_pb.Code_MARSHAL_FAILED
		return &rslt
	}

	return &rslt
}

func (player *Player) Update(ctx context.Context, args *rkcy.StepArgs) *rkcy.StepResult {
	log.Info().
		Msg("storage/player.go/Update")
	return nil
}

func (player *Player) Delete(ctx context.Context, args *rkcy.StepArgs) *rkcy.StepResult {
	log.Info().
		Msg("storage/player.go/Delete")
	return nil
}
