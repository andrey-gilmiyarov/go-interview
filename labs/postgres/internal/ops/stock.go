package ops

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Buy(ctx context.Context, p *pgxpool.Pool, product int64, quantity int) (int64, error) {
	if quantity <= 0 {
		return 0, fmt.Errorf("quantity must be positive")
	}
	tx, err := p.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer Rollback(tx)
	tag, err := tx.Exec(ctx, "UPDATE products SET stock=stock-$2 WHERE id=$1 AND stock >= $2", product, quantity)
	if err != nil {
		return 0, err
	}
	if tag.RowsAffected() != 1 {
		return 0, ErrStock
	}
	var id int64
	if err = tx.QueryRow(ctx, "INSERT INTO orders(product_id,quantity) VALUES($1,$2) RETURNING id", product, quantity).Scan(&id); err != nil {
		return 0, err
	}
	return id, tx.Commit(ctx)
}
