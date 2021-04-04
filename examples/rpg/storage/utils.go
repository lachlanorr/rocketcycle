package storage

import (
	"context"
	"github.com/jackc/pgx/v4"
)

func connect(ctx context.Context) (*pgx.Conn, error) {
	// LORRTODO: make connection string a config
	return pgx.Connect(ctx, "postgresql://postgres@127.0.0.1:5432/rpg")
}
