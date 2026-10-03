//go:build integration

package lab

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMigrationAndBulkExamples(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	err := WithSchema(ctx, "query-plan", func(p *pgxpool.Pool) error {
		conn, e := p.Acquire(ctx)
		if e != nil {
			return e
		}
		defer conn.Release()
		source, e := os.ReadFile("../../examples/migration.sql")
		if e != nil {
			return e
		}
		// Statements are intentionally outside a transaction: CONCURRENTLY requires it.
		for _, stmt := range strings.Split(string(source), ";") {
			if strings.TrimSpace(stmt) != "" {
				if _, e = conn.Exec(ctx, stmt); e != nil {
					return e
				}
			}
		}
		var valid bool
		if e = conn.QueryRow(ctx, "SELECT indisvalid FROM pg_index WHERE indexrelid='migration_orders_customer'::regclass").Scan(&valid); e != nil {
			return e
		}
		if !valid {
			t.Error("invalid concurrent index")
		}
		return BulkExample(ctx, p)
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestScenarioLockAndCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	err := WithSchema(ctx, "deadlock", func(p *pgxpool.Pool) error {
		if e := WithSchema(ctx, "deadlock", func(*pgxpool.Pool) error { t.Error("concurrent setup entered"); return nil }); e == nil {
			t.Error("lock not enforced")
		}
		a, e := p.Begin(ctx)
		if e != nil {
			return e
		}
		defer rollback(a)
		if _, e = a.Exec(ctx, "UPDATE products SET stock=stock WHERE id=1"); e != nil {
			return e
		}
		b, e := p.Begin(ctx)
		if e != nil {
			return e
		}
		defer rollback(b)
		waiting, stop := context.WithCancel(ctx)
		ch := make(chan error, 1)
		go func() { _, e := b.Exec(waiting, "SELECT id FROM products WHERE id=1 FOR UPDATE"); ch <- e }()
		if e = waitBlocked(ctx, p, b.Conn().PgConn().PID()); e != nil {
			stop()
			<-ch
			return e
		}
		stop()
		if e = <-ch; e == nil {
			t.Error("blocked statement ignored cancellation")
		}
		rollback(b)
		rollback(a)
		tx, e := p.Begin(ctx)
		if e != nil {
			return e
		}
		defer rollback(tx)
		// Cancellation may asynchronously close the old backend. The bounded
		// lock_timeout proves eventual release without an arbitrary sleep.
		_, e = tx.Exec(ctx, "SELECT id FROM products WHERE id=1 FOR UPDATE")
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Compile-time check that fixtures retain the native pgx transaction API.
var _ pgx.TxIsoLevel = pgx.Serializable
