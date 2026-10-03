package solution

import (
	"context"

	"github.com/andreygilmiyarov/go-interview/labs/postgres/internal/ops"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Buy(ctx context.Context, p *pgxpool.Pool, product int64, quantity int) (int64, error) {
	return ops.Buy(ctx, p, product, quantity)
}
