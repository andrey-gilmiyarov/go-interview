package lab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/andreygilmiyarov/go-interview/labs/postgres/internal/ops"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func queryPlan(ctx context.Context, p *pgxpool.Pool, w io.Writer) error {
	if _, err := p.Exec(ctx, `CREATE TABLE history(id bigint PRIMARY KEY,customer_id integer NOT NULL,created_at timestamptz NOT NULL,payload text);
 INSERT INTO history SELECT n,n%1000,timestamptz '2026-01-01 00:00:00+00'+n*interval '1 second',repeat('x',100) FROM generate_series(1,100000) n; ANALYZE history`); err != nil {
		return err
	}
	query := "SELECT id FROM history WHERE customer_id=42 ORDER BY created_at DESC LIMIT 20"
	read := func() ([]int64, error) {
		rows, e := p.Query(ctx, query)
		if e != nil {
			return nil, e
		}
		return pgx.CollectRows(rows, pgx.RowTo[int64])
	}
	before, err := read()
	if err != nil {
		return err
	}
	for _, phase := range []string{"before", "after"} {
		if phase == "after" {
			if _, err = p.Exec(ctx, "CREATE INDEX history_customer_created ON history(customer_id,created_at DESC) INCLUDE(id); ANALYZE history"); err != nil {
				return err
			}
		}
		var plan json.RawMessage
		if err = p.QueryRow(ctx, "EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) "+query).Scan(&plan); err != nil {
			return err
		}
		if !json.Valid(plan) {
			return fmt.Errorf("invalid plan")
		}
		fmt.Fprintf(w, "%s (100000 rows): %s\n", phase, plan)
	}
	after, err := read()
	if err != nil {
		return err
	}
	return require(fmt.Sprint(before) == fmt.Sprint(after) && len(before) == 20, "query results changed")
}
func jobQueue(ctx context.Context, p *pgxpool.Pool, w io.Writer) error {
	if _, err := p.Exec(ctx, `CREATE TABLE jobs(id bigint PRIMARY KEY,token bigint NOT NULL DEFAULT 0,lease_until timestamptz,done boolean NOT NULL DEFAULT false); INSERT INTO jobs(id) VALUES(1),(2)`); err != nil {
		return err
	}
	type result struct {
		j ops.Job
		e error
	}
	ch := make(chan result, 2)
	for range 2 {
		go func() { j, e := ops.Claim(ctx, p); ch <- result{j, e} }()
	}
	a, b := <-ch, <-ch
	if a.e != nil {
		return a.e
	}
	if b.e != nil {
		return b.e
	}
	if a.j.ID == b.j.ID {
		return fmt.Errorf("double claim")
	}
	if _, err := p.Exec(ctx, "UPDATE jobs SET lease_until=clock_timestamp()-interval '1 second' WHERE id=$1", a.j.ID); err != nil {
		return err
	}
	replacement, err := ops.Claim(ctx, p)
	if err != nil {
		return err
	}
	if replacement.ID != a.j.ID || replacement.Token <= a.j.Token {
		return fmt.Errorf("lease recovery failed")
	}
	ok, err := ops.Ack(ctx, p, a.j)
	if err != nil {
		return err
	}
	if ok {
		return fmt.Errorf("stale worker acknowledged")
	}
	for _, j := range []ops.Job{replacement, b.j} {
		ok, err = ops.Ack(ctx, p, j)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("current worker rejected")
		}
	}
	_, err = ops.Claim(ctx, p)
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("expected empty queue: %v", err)
	}
	fmt.Fprintln(w, "Distinct claims; expired lease reclaimed; stale token rejected. External side effects still require idempotency.")
	return nil
}
func longTransaction(ctx context.Context, p *pgxpool.Pool, w io.Writer) error {
	if _, err := p.Exec(ctx, "CREATE TABLE versions(id int PRIMARY KEY,value int) WITH (autovacuum_enabled=false); INSERT INTO versions SELECT n,0 FROM generate_series(1,1000)n"); err != nil {
		return err
	}
	tx, err := p.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return err
	}
	defer rollback(tx)
	var sum int
	if err = tx.QueryRow(ctx, "SELECT sum(value) FROM versions").Scan(&sum); err != nil {
		return err
	}
	if _, err = p.Exec(ctx, "UPDATE versions SET value=1"); err != nil {
		return err
	}
	cfg := p.Config().ConnConfig.Copy()
	cfg.OnNotice = func(_ *pgconn.PgConn, n *pgconn.Notice) { fmt.Fprintln(w, n.Message, n.Detail) }
	vacuum, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer cleanup(vacuum.Close)
	if _, err = vacuum.Exec(ctx, "VACUUM (VERBOSE, ANALYZE) versions"); err != nil {
		return err
	}
	if err = tx.QueryRow(ctx, "SELECT sum(value) FROM versions").Scan(&sum); err != nil {
		return err
	}
	if sum != 0 {
		return fmt.Errorf("old snapshot changed")
	}
	var pinned bool
	if err = p.QueryRow(ctx, "SELECT backend_xmin IS NOT NULL FROM pg_stat_activity WHERE pid=$1", int64(tx.Conn().PgConn().PID())).Scan(&pinned); err != nil {
		return err
	}
	if !pinned {
		return fmt.Errorf("snapshot not pinned")
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	if _, err = vacuum.Exec(ctx, "VACUUM (VERBOSE, ANALYZE) versions"); err != nil {
		return err
	}
	if err = p.QueryRow(ctx, "SELECT sum(value) FROM versions").Scan(&sum); err != nil {
		return err
	}
	fmt.Fprintf(w, "Snapshot released; new sum=%d; compare VERBOSE removal reports, not relation size\n", sum)
	return require(sum == 1000, "new snapshot missing updates")
}
func goPool(ctx context.Context, p *pgxpool.Pool, w io.Writer) error {
	cfg := p.Config()
	cfg.MaxConns = 1
	small, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer small.Close()
	held, err := small.Acquire(ctx)
	if err != nil {
		return err
	}
	wait, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	_, err = small.Acquire(wait)
	cancel()
	held.Release()
	if !errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("expected pool deadline: %v", err)
	}
	queryCtx, stop := context.WithTimeout(ctx, 50*time.Millisecond)
	_, err = small.Exec(queryCtx, "SELECT pg_sleep(10)")
	stop()
	if err == nil {
		return fmt.Errorf("query cancellation missing")
	}
	stocks, err := ops.ReadStocks(ctx, small)
	if err != nil {
		return err
	}
	if len(stocks) != 2 {
		return fmt.Errorf("missing stocks")
	}
	tx, err := small.Begin(ctx)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "SELECT 1/0")
	if code(err) != "22012" {
		rollback(tx)
		return fmt.Errorf("expected query failure: %v", err)
	}
	rollback(tx)
	if err = small.Ping(ctx); err != nil {
		return err
	}
	if small.Stat().AcquiredConns() != 0 {
		return fmt.Errorf("connection leak")
	}
	fmt.Fprintln(w, "Acquire deadline, query cancellation, rows close and rollback verified; pool reusable")
	return nil
}
