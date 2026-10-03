// Package checks shares the real-database contract between starter and solution.
package checks

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andreygilmiyarov/go-interview/labs/postgres/internal/lab"
	"github.com/andreygilmiyarov/go-interview/labs/postgres/internal/ops"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func run(t *testing.T, id string, fn func(context.Context, *pgxpool.Pool) error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := lab.WithSchema(ctx, id, func(p *pgxpool.Pool) error { return fn(ctx, p) }); err != nil {
		t.Fatal(err)
	}
}

// Exercise checks have separate identifiers from the automated scenario tests;
// packages run serially in verify.sh to avoid competing for the same schema.
func Buy(t *testing.T, fn func(context.Context, *pgxpool.Pool, int64, int) (int64, error)) {
	run(t, "stock-race", func(ctx context.Context, p *pgxpool.Pool) error {
		ch := make(chan error, 2)
		for range 2 {
			go func() { _, e := fn(ctx, p, 1, 1); ch <- e }()
		}
		a, b := <-ch, <-ch
		if !((a == nil && errors.Is(b, ops.ErrStock)) || (b == nil && errors.Is(a, ops.ErrStock))) {
			t.Errorf("want one purchase, one ErrStock: %v / %v", a, b)
		}
		var n, stock int
		if e := p.QueryRow(ctx, "SELECT count(*),(SELECT stock FROM products WHERE id=1) FROM orders").Scan(&n, &stock); e != nil {
			return e
		}
		if n != 1 || stock != 0 {
			t.Errorf("orders=%d stock=%d", n, stock)
		}
		if _, e := fn(ctx, p, 2, 0); e == nil {
			t.Error("zero quantity accepted")
		}
		// Force an INSERT failure AFTER a successful stock update; stock must roll back.
		if _, e := p.Exec(ctx, "ALTER TABLE orders ADD CHECK(product_id<>2)"); e != nil {
			return e
		}
		if _, e := fn(ctx, p, 2, 1); e == nil {
			t.Error("constraint failure swallowed")
		}
		if e := p.QueryRow(ctx, "SELECT stock FROM products WHERE id=2").Scan(&stock); e != nil {
			return e
		}
		if stock != 1 {
			t.Error("stock changed despite insert failure")
		}
		return nil
	})
}
func Idempotency(t *testing.T, fn func(context.Context, *pgxpool.Pool, string, int64, int) (int64, error)) {
	run(t, "idempotency", func(ctx context.Context, p *pgxpool.Pool) error {
		type result struct {
			id int64
			e  error
		}
		ch := make(chan result, 2)
		for range 2 {
			go func() { id, e := fn(ctx, p, "key", 1, 1); ch <- result{id, e} }()
		}
		a, b := <-ch, <-ch
		if a.e != nil {
			return a.e
		}
		if b.e != nil {
			return b.e
		}
		if a.id != b.id {
			t.Error("duplicate result")
		}
		if _, e := fn(ctx, p, "key", 2, 1); !errors.Is(e, ops.ErrConflict) {
			t.Errorf("payload conflict: %v", e)
		}
		if _, e := fn(ctx, p, "empty", 1, 1); !errors.Is(e, ops.ErrStock) {
			t.Errorf("stock: %v", e)
		}
		var count int
		if e := p.QueryRow(ctx, "SELECT count(*) FROM orders").Scan(&count); e != nil {
			return e
		}
		if count != 1 {
			t.Errorf("failed order persisted: %d", count)
		}
		return nil
	})
}
func Queue(t *testing.T, claim func(context.Context, *pgxpool.Pool) (ops.Job, error), ack func(context.Context, *pgxpool.Pool, ops.Job) (bool, error)) {
	run(t, "job-queue", func(ctx context.Context, p *pgxpool.Pool) error {
		if _, e := p.Exec(ctx, `CREATE TABLE jobs(id bigint PRIMARY KEY,token bigint NOT NULL DEFAULT 0,lease_until timestamptz,done boolean NOT NULL DEFAULT false); INSERT INTO jobs(id) VALUES(1),(2)`); e != nil {
			return e
		}
		tx, e := p.Begin(ctx)
		if e != nil {
			return e
		}
		defer ops.Rollback(tx)
		if _, e = tx.Exec(ctx, "SELECT id FROM jobs WHERE id=1 FOR UPDATE"); e != nil {
			return e
		}
		j, e := claim(ctx, p)
		if e != nil {
			return e
		}
		if j.ID != 2 {
			t.Errorf("SKIP LOCKED claimed %d", j.ID)
		}
		ops.Rollback(tx)
		if _, e = p.Exec(ctx, "UPDATE jobs SET lease_until=clock_timestamp()-interval '1 second' WHERE id=2; DELETE FROM jobs WHERE id=1"); e != nil {
			return e
		}
		next, e := claim(ctx, p)
		if e != nil {
			return e
		}
		if next.ID != j.ID || next.Token <= j.Token {
			t.Error("claim did not fence old worker")
		}
		if ok, e := ack(ctx, p, j); e != nil {
			return e
		} else if ok {
			t.Error("stale acknowledgement accepted")
		}
		if ok, e := ack(ctx, p, next); e != nil {
			return e
		} else if !ok {
			t.Error("current acknowledgement rejected")
		}
		if _, e = claim(ctx, p); !errors.Is(e, pgx.ErrNoRows) {
			t.Errorf("empty queue: %v", e)
		}
		return nil
	})
}
func Pool(t *testing.T, fn func(context.Context, *pgxpool.Pool) ([]int, error)) {
	run(t, "go-pool", func(ctx context.Context, p *pgxpool.Pool) error {
		cfg := p.Config()
		cfg.MaxConns = 1
		small, e := pgxpool.NewWithConfig(ctx, cfg)
		if e != nil {
			return e
		}
		defer small.Close()
		values, e := fn(ctx, small)
		if e != nil {
			return e
		}
		if len(values) != 2 {
			t.Error("expected two rows")
		}
		c, cancel := context.WithCancel(ctx)
		cancel()
		if _, e = fn(c, small); e == nil {
			t.Error("cancel ignored")
		}
		if _, e = small.Exec(ctx, "DROP TABLE orders; DROP TABLE products"); e != nil {
			return e
		}
		if _, e = fn(ctx, small); e == nil {
			t.Error("query error swallowed")
		}
		if e = small.Ping(ctx); e != nil {
			return e
		}
		if small.Stat().AcquiredConns() != 0 {
			t.Error("connection leaked")
		}
		return nil
	})
}
