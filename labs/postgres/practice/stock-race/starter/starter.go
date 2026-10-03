package starter

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Buy(ctx context.Context, p *pgxpool.Pool, product int64, quantity int) (int64, error) {
	return 0, errors.New("implement me")
}
