package ops

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Job struct{ ID, Token int64 }

func Claim(ctx context.Context, p *pgxpool.Pool) (Job, error) {
	var j Job
	err := p.QueryRow(ctx, `WITH candidate AS (
 SELECT id FROM jobs WHERE NOT done AND (lease_until IS NULL OR lease_until < clock_timestamp())
 ORDER BY id FOR UPDATE SKIP LOCKED LIMIT 1)
 UPDATE jobs SET token=token+1,lease_until=clock_timestamp()+interval '30 seconds'
 FROM candidate WHERE jobs.id=candidate.id RETURNING jobs.id,jobs.token`).Scan(&j.ID, &j.Token)
	return j, err
}
func Ack(ctx context.Context, p *pgxpool.Pool, j Job) (bool, error) {
	tag, err := p.Exec(ctx, `UPDATE jobs SET done=true WHERE id=$1 AND token=$2 AND NOT done AND lease_until>clock_timestamp()`, j.ID, j.Token)
	return tag.RowsAffected() == 1, err
}
