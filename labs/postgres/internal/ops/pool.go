package ops

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ReadStocks(ctx context.Context, p *pgxpool.Pool) ([]int, error) {
	tx, err := p.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer Rollback(tx)
	rows, err := tx.Query(ctx, "SELECT stock FROM products ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stocks []int
	for rows.Next() {
		var n int
		if err = rows.Scan(&n); err != nil {
			return nil, err
		}
		stocks = append(stocks, n)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return stocks, tx.Commit(ctx)
}
