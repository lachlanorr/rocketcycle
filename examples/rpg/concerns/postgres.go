package concerns

import (
	"context"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog/log"
)

var pool *pgxpool.Pool

func init() {
	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		connString = "postgresql://postgres@127.0.0.1:5432/rpg"
	}

	var err error
	pool, err = pgxpool.Connect(context.Background(), connString)
	if err != nil {
		log.Fatal().
			Err(err).
			Msgf("Failed to create pgxpool: %s", connString)
	}
}
