package ops

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateOrder(ctx context.Context, p *pgxpool.Pool, key string, product int64, quantity int) (int64, error) {
	if key == "" || quantity <= 0 {
		return 0, fmt.Errorf("key and positive quantity required")
	}
	tx, err := p.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer Rollback(tx)
	var id int64
	err = tx.QueryRow(ctx, `INSERT INTO orders(request_key,product_id,quantity) VALUES($1,$2,$3)
 ON CONFLICT(request_key) DO NOTHING RETURNING id`, key, product, quantity).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		var existingProduct int64
		var existingQuantity int
		// Separate statement under Read Committed sees the committed competing row.
		if err = tx.QueryRow(ctx, "SELECT id,product_id,quantity FROM orders WHERE request_key=$1", key).Scan(&id, &existingProduct, &existingQuantity); err != nil {
			return 0, err
		}
		if existingProduct != product || existingQuantity != quantity {
			return 0, ErrConflict
		}
	} else if err != nil {
		return 0, err
	} else {
		tag, err := tx.Exec(ctx, "UPDATE products SET stock=stock-$2 WHERE id=$1 AND stock >= $2", product, quantity)
		if err != nil {
			return 0, err
		}
		if tag.RowsAffected() != 1 {
			return 0, ErrStock
		}
	}
	return id, tx.Commit(ctx)
}
