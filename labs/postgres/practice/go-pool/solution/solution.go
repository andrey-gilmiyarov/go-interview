package solution

import (
	"context"

	"github.com/andreygilmiyarov/go-interview/labs/postgres/internal/ops"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ReadStocks(ctx context.Context, p *pgxpool.Pool) ([]int, error) { return ops.ReadStocks(ctx, p) }
