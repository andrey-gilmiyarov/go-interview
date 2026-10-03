package starter

import (
	"context"
	"errors"

	"github.com/andreygilmiyarov/go-interview/labs/postgres/internal/ops"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Claim(ctx context.Context, p *pgxpool.Pool) (ops.Job, error) {
	return ops.Job{}, errors.New("implement me")
}
func Ack(ctx context.Context, p *pgxpool.Pool, j ops.Job) (bool, error) {
	return false, errors.New("implement me")
}
