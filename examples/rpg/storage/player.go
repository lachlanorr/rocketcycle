// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package storage

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"

	rpg_pb "github.com/lachlanorr/rocketcycle/examples/rpg/pb"
	rkcy_pb "github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
)

func connect() (*pgx.Conn, error) {
	// LORRTODO: make connection string a config
	return pgx.Connect(context.Background(), "postgresql://postgres@127.0.0.1:5432/rpg")
}

type Player struct {
}

func (player *Player) Get(uid string, id string) *rkcy_pb.ApecsStorageResult {
	return nil
}

func (player *Player) Create(uid string, id string, payload []byte, mro *rkcy_pb.MostRecentOffset) *rkcy_pb.ApecsStorageResult {
	mdl := rpg_pb.Player{}
	err := proto.Unmarshal(payload, &mdl)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to Unmarshal Player")
		return &rkcy_pb.ApecsStorageResult{
			Uid:  uid,
			Code: rkcy_pb.ApecsStorageResult_MARSHAL_FAILED,
		}
	}

	conn, err := connect()
	if err != nil {
		log.Error().
			Err(err).
			Msg("Unable to connect to database")
		return &rkcy_pb.ApecsStorageResult{
			Uid:  uid,
			Code: rkcy_pb.ApecsStorageResult_CONNECTION,
		}
	}
	defer conn.Close(context.Background())

	stmt := `INSERT INTO rpg.player (id, username, active, mro_generation, mro_offset)
             VALUES ($1, $2, $3, $4, $5)
             RETURNING id`

	retid := ""
	err = conn.QueryRow(
		context.Background(),
		stmt,
		mdl.Id,
		mdl.Username,
		mdl.Active,
		mro.Generation,
		mro.Offset,
	).Scan(&retid)

	if err != nil {
		log.Error().
			Err(err).
			Msg("Unable to create rpg.player")
		return &rkcy_pb.ApecsStorageResult{
			Uid:  uid,
			Code: rkcy_pb.ApecsStorageResult_INTERNAL,
		}
	}

	mdlSer, err := proto.Marshal(&mdl)
	if err != nil {
		log.Error().
			Err(err).
			Msg("Failed to Marshal Player")
		return &rkcy_pb.ApecsStorageResult{
			Uid:  uid,
			Code: rkcy_pb.ApecsStorageResult_MARSHAL_FAILED,
		}
	}

	return &rkcy_pb.ApecsStorageResult{
		Uid:     uid,
		Code:    rkcy_pb.ApecsStorageResult_SUCCESS,
		Payload: mdlSer,
	}
}

func (player *Player) Update(uid string, id string, payload []byte, mro *rkcy_pb.MostRecentOffset) *rkcy_pb.ApecsStorageResult {
	return nil
}

func (player *Player) Delete(uid string, id string, mro *rkcy_pb.MostRecentOffset) *rkcy_pb.ApecsStorageResult {
	return nil
}
