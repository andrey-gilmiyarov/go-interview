package lab

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BulkExample demonstrates COPY and draining a batch within a transaction.
func BulkExample(ctx context.Context, p *pgxpool.Pool) error {
	tx, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if _, err = tx.Exec(ctx, "CREATE TEMP TABLE import_items(id bigint PRIMARY KEY,label text) ON COMMIT DROP"); err != nil {
		return err
	}
	n, err := tx.CopyFrom(ctx, pgx.Identifier{"import_items"}, []string{"id", "label"}, pgx.CopyFromRows([][]any{{int64(1), "first"}, {int64(2), "second"}}))
	if err != nil {
		return err
	}
	if n != 2 {
		return fmt.Errorf("COPY inserted %d", n)
	}
	batch := &pgx.Batch{}
	batch.Queue("UPDATE import_items SET label=$1 WHERE id=$2", "changed", int64(1))
	batch.Queue("SELECT count(*) FROM import_items")
	result := tx.SendBatch(ctx, batch)
	if _, err = result.Exec(); err != nil {
		_ = result.Close()
		return err
	}
	var count int
	err = result.QueryRow().Scan(&count)
	closeErr := result.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if count != 2 {
		return fmt.Errorf("batch count %d", count)
	}
	return tx.Commit(ctx)
}
