package starter

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ReadStocks(ctx context.Context, p *pgxpool.Pool) ([]int, error) {
	return nil, errors.New("implement me")
}
