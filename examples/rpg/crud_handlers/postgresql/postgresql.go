package postgresql

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v4/pgxpool"
)

var pool *pgxpool.Pool = nil
var poolMtx sync.Mutex

func InitPostgresqlPool(ctx context.Context, wg *sync.WaitGroup, config map[string]string) error {
	poolMtx.Lock()
	defer poolMtx.Unlock()

	if pool == nil {
		connString, ok := config["connString"]
		if !ok {
			return fmt.Errorf("No connString specified in config")
		}

		var err error
		pool, err = pgxpool.Connect(context.Background(), connString)
		if err != nil {
			return fmt.Errorf("Failed to create pgxpool: %s", connString)
		}
	}

	return nil
}
