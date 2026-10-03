package solution

import (
	"context"

	"github.com/andreygilmiyarov/go-interview/labs/postgres/internal/ops"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Claim(ctx context.Context, p *pgxpool.Pool) (ops.Job, error)       { return ops.Claim(ctx, p) }
func Ack(ctx context.Context, p *pgxpool.Pool, j ops.Job) (bool, error) { return ops.Ack(ctx, p, j) }
