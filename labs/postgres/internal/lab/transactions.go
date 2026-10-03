package lab

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/andreygilmiyarov/go-interview/labs/postgres/internal/ops"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func stockRace(ctx context.Context, p *pgxpool.Pool, w io.Writer) error {
	// Both reads precede either write: deterministic lost update, no sleep.
	var a, b int
	if err := p.QueryRow(ctx, "SELECT stock FROM products WHERE id=1").Scan(&a); err != nil {
		return err
	}
	if err := p.QueryRow(ctx, "SELECT stock FROM products WHERE id=1").Scan(&b); err != nil {
		return err
	}
	for _, n := range []int{a, b} {
		if _, err := p.Exec(ctx, "UPDATE products SET stock=$1 WHERE id=1", n-1); err != nil {
			return err
		}
		if _, err := p.Exec(ctx, "INSERT INTO orders(product_id,quantity) VALUES(1,1)"); err != nil {
			return err
		}
	}
	fmt.Fprintln(w, "Broken: two orders sold from stock=1; CHECK(stock>=0) did not protect the cross-table invariant")
	if _, err := p.Exec(ctx, "TRUNCATE orders; UPDATE products SET stock=1 WHERE id=1"); err != nil {
		return err
	}
	results := make(chan error, 2)
	start := make(chan struct{})
	for range 2 {
		go func() { <-start; _, err := ops.Buy(ctx, p, 1, 1); results <- err }()
	}
	close(start)
	successes := 0
	for range 2 {
		err := <-results
		if err == nil {
			successes++
		} else if !errors.Is(err, ops.ErrStock) {
			return err
		}
	}
	var stock, count int
	if err := p.QueryRow(ctx, "SELECT stock,(SELECT count(*) FROM orders) FROM products WHERE id=1").Scan(&stock, &count); err != nil {
		return err
	}
	fmt.Fprintf(w, "Fixed: successes=%d orders=%d stock=%d\n", successes, count, stock)
	return require(successes == 1 && count == 1 && stock == 0, "stock invariant failed")
}
func isolation(ctx context.Context, p *pgxpool.Pool, w io.Writer) error {
	for _, level := range []pgx.TxIsoLevel{pgx.ReadCommitted, pgx.RepeatableRead} {
		if _, err := p.Exec(ctx, "UPDATE products SET stock=1"); err != nil {
			return err
		}
		tx, err := p.BeginTx(ctx, pgx.TxOptions{IsoLevel: level})
		if err != nil {
			return err
		}
		defer rollback(tx)
		var before, after int
		if err = tx.QueryRow(ctx, "SELECT stock FROM products WHERE id=1").Scan(&before); err != nil {
			return err
		}
		if _, err = p.Exec(ctx, "UPDATE products SET stock=2 WHERE id=1"); err != nil {
			return err
		}
		if err = tx.QueryRow(ctx, "SELECT stock FROM products WHERE id=1").Scan(&after); err != nil {
			return err
		}
		if err = tx.Commit(ctx); err != nil {
			return err
		}
		expected := 1
		if level == pgx.ReadCommitted {
			expected = 2
		}
		if err = require(before == 1 && after == expected, "visibility %s: %d -> %d", level, before, after); err != nil {
			return err
		}
		fmt.Fprintf(w, "%s: %d -> %d\n", level, before, after)
	}
	for _, level := range []pgx.TxIsoLevel{pgx.RepeatableRead, pgx.Serializable} {
		if _, err := p.Exec(ctx, "UPDATE products SET stock=1"); err != nil {
			return err
		}
		a, err := p.BeginTx(ctx, pgx.TxOptions{IsoLevel: level})
		if err != nil {
			return err
		}
		defer rollback(a)
		b, err := p.BeginTx(ctx, pgx.TxOptions{IsoLevel: level})
		if err != nil {
			return err
		}
		defer rollback(b)
		var total int
		for _, tx := range []pgx.Tx{a, b} {
			if err = tx.QueryRow(ctx, "SELECT sum(stock) FROM products").Scan(&total); err != nil {
				return err
			}
			if total != 2 {
				return fmt.Errorf("bad fixture")
			}
		}
		if _, err = a.Exec(ctx, "UPDATE products SET stock=0 WHERE id=1"); err != nil {
			return err
		}
		if _, err = b.Exec(ctx, "UPDATE products SET stock=0 WHERE id=2"); err != nil {
			return err
		}
		ea, eb := a.Commit(ctx), b.Commit(ctx)
		if level == pgx.RepeatableRead {
			if ea != nil {
				return ea
			}
			if eb != nil {
				return eb
			}
		} else {
			if !((ea == nil && code(eb) == "40001") || (eb == nil && code(ea) == "40001")) {
				return fmt.Errorf("expected one serialization abort: %v / %v", ea, eb)
			}
			retryID := int64(2)
			if ea != nil {
				retryID = 1
			}
			err = Retry(ctx, 3, func(ctx context.Context) error {
				tx, e := p.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
				if e != nil {
					return e
				}
				defer rollback(tx)
				var n int
				if e = tx.QueryRow(ctx, "SELECT sum(stock) FROM products").Scan(&n); e != nil {
					return e
				}
				if n > 1 {
					if _, e = tx.Exec(ctx, "UPDATE products SET stock=0 WHERE id=$1", retryID); e != nil {
						return e
					}
				}
				return tx.Commit(ctx)
			})
			if err != nil {
				return err
			}
		}
		if err = p.QueryRow(ctx, "SELECT sum(stock) FROM products").Scan(&total); err != nil {
			return err
		}
		expected := 0
		if level == pgx.Serializable {
			expected = 1
		}
		fmt.Fprintf(w, "%s write skew: remaining=%d aborts=%s/%s\n", level, total, code(ea), code(eb))
		if total != expected {
			return fmt.Errorf("write skew invariant")
		}
	}
	return nil
}
func deadlock(ctx context.Context, p *pgxpool.Pool, w io.Writer) error {
	a, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(a)
	b, err := p.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollback(b)
	if _, err = a.Exec(ctx, "UPDATE products SET stock=stock WHERE id=1"); err != nil {
		return err
	}
	if _, err = b.Exec(ctx, "UPDATE products SET stock=stock WHERE id=2"); err != nil {
		return err
	}
	ch := make(chan error, 1)
	go func() {
		_, e := a.Exec(ctx, "UPDATE products SET stock=stock WHERE id=2")
		if e != nil {
			rollback(a)
		}
		ch <- e
	}()
	if err = waitBlocked(ctx, p, a.Conn().PgConn().PID()); err != nil {
		rollback(b)
		<-ch
		return err
	}
	fmt.Fprintf(w, "Blocked PID %d waits for %d\n", a.Conn().PgConn().PID(), b.Conn().PgConn().PID())
	_, eb := b.Exec(ctx, "UPDATE products SET stock=stock WHERE id=1")
	if eb != nil {
		rollback(b)
	}
	ea := <-ch
	if !((code(ea) == "40P01" && eb == nil) || (code(eb) == "40P01" && ea == nil)) {
		return fmt.Errorf("expected deadlock: %v / %v", ea, eb)
	}
	rollback(a)
	rollback(b)
	results := make(chan error, 2)
	for range 2 {
		go func() {
			tx, e := p.Begin(ctx)
			if e != nil {
				results <- e
				return
			}
			defer rollback(tx)
			for _, id := range []int{1, 2} {
				if _, e = tx.Exec(ctx, "UPDATE products SET stock=stock WHERE id=$1", id); e != nil {
					results <- e
					return
				}
			}
			results <- tx.Commit(ctx)
		}()
	}
	var final error
	for range 2 {
		final = errors.Join(final, <-results)
	}
	fmt.Fprintln(w, "Consistent order: both transactions complete")
	return final
}
func idempotency(ctx context.Context, p *pgxpool.Pool, w io.Writer) error {
	// Both clients observe an absent key before either inserts it.
	for range 2 {
		var count int
		if err := p.QueryRow(ctx, "SELECT count(*) FROM orders WHERE request_key='naive'").Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return fmt.Errorf("invalid naive fixture")
		}
	}
	if _, err := p.Exec(ctx, "INSERT INTO orders(request_key,product_id,quantity) VALUES('naive',1,1)"); err != nil {
		return err
	}
	_, naiveErr := p.Exec(ctx, "INSERT INTO orders(request_key,product_id,quantity) VALUES('naive',1,1)")
	if code(naiveErr) != "23505" {
		return fmt.Errorf("expected check-before-insert conflict: %v", naiveErr)
	}
	fmt.Fprintln(w, "Broken: both SELECTs found no key; second INSERT fails with 23505 instead of returning the existing order")
	if _, err := p.Exec(ctx, "TRUNCATE orders"); err != nil {
		return err
	}
	type result struct {
		id  int64
		err error
	}
	ch := make(chan result, 2)
	for range 2 {
		go func() { id, e := ops.CreateOrder(ctx, p, "request-1", 1, 1); ch <- result{id, e} }()
	}
	a, b := <-ch, <-ch
	if a.err != nil {
		return a.err
	}
	if b.err != nil {
		return b.err
	}
	if a.id != b.id {
		return fmt.Errorf("duplicate order")
	}
	_, err := ops.CreateOrder(ctx, p, "request-1", 1, 2)
	if !errors.Is(err, ops.ErrConflict) {
		return fmt.Errorf("expected payload conflict: %v", err)
	}
	var count, stock int
	if err = p.QueryRow(ctx, "SELECT stock,(SELECT count(*) FROM orders) FROM products WHERE id=1").Scan(&stock, &count); err != nil {
		return err
	}
	fmt.Fprintf(w, "Same key -> order %d, orders=%d stock=%d; changed payload rejected\n", a.id, count, stock)
	return require(count == 1 && stock == 0, "idempotency invariant failed")
}
