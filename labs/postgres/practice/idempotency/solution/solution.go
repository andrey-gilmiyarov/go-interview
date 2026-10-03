package solution

import (
	"context"

	"github.com/andreygilmiyarov/go-interview/labs/postgres/internal/ops"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateOrder(ctx context.Context, p *pgxpool.Pool, key string, product int64, quantity int) (int64, error) {
	return ops.CreateOrder(ctx, p, key, product, quantity)
}
