package pb

import (
	"context"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"
)

var _pool *pgxpool.Pool = nil

func pool() *pgxpool.Pool {
	if _pool == nil {
		connString := os.Getenv("DATABASE_URL")
		if connString == "" {
			connString = "postgresql://postgres@127.0.0.1:5432/rpg"
		}

		var err error
		_pool, err = pgxpool.Connect(context.Background(), connString)
		if err != nil {
			log.Fatal().
				Err(err).
				Msgf("Failed to create pgxpool: %s", connString)
		}
	}
	return _pool
}
