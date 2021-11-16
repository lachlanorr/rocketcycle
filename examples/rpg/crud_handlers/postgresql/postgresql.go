package postgresql

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v4/pgxpool"
)

var pool *pgxpool.Pool = nil

func InitPostgresqlPool(ctx context.Context, config map[string]string, wg *sync.WaitGroup) error {
	connString, ok := config["connString"]
	if !ok {
		return fmt.Errorf("No connString specified in config")
	}

	var err error
	pool, err = pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return fmt.Errorf("Failed to create pgxpool: %s", connString)
	}

	return nil
}
