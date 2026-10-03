package lab

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var IDs = []string{"stock-race", "isolation", "deadlock", "idempotency", "query-plan", "job-queue", "long-transaction", "go-pool"}

// Config intentionally does not read DATABASE_URL or PGHOST: this lab only
// modifies its own local database. Port is the only connection override.
func Config() (*pgxpool.Config, error) {
	port := os.Getenv("POSTGRES_LAB_PORT")
	if port == "" {
		port = "45432"
	}
	n, err := strconv.Atoi(port)
	if err != nil || n <= 1024 || n > 65535 {
		return nil, fmt.Errorf("invalid POSTGRES_LAB_PORT")
	}
	cfg, err := pgxpool.ParseConfig("postgres://postgres_lab:postgres_lab@127.0.0.1:" + port + "/postgres_lab?sslmode=disable")
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 8
	cfg.ConnConfig.ConnectTimeout = 3 * time.Second
	cfg.ConnConfig.RuntimeParams = map[string]string{"application_name": "postgres-handbook", "statement_timeout": "15000", "lock_timeout": "5000", "idle_in_transaction_session_timeout": "20000"}
	return cfg, nil
}

func cleanup(fn func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = fn(ctx)
}
func rollback(tx pgx.Tx) { cleanup(tx.Rollback) }
func code(err error) string {
	var e *pgconn.PgError
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
func require(ok bool, format string, args ...any) error {
	if !ok {
		return fmt.Errorf(format, args...)
	}
	return nil
}

// WithSchema holds a session advisory lock across schema setup and the callback.
// Tables use a scenario-specific search_path on EVERY pool connection.
func WithSchema(ctx context.Context, id string, fn func(*pgxpool.Pool) error) error {
	valid := false
	var key int64
	for i, v := range IDs {
		if v == id {
			valid = true
			key = int64(731000 + i)
		}
	}
	if !valid {
		return fmt.Errorf("unknown scenario %q", id)
	}
	cfg, err := Config()
	if err != nil {
		return err
	}
	guard, err := pgx.ConnectConfig(ctx, cfg.ConnConfig.Copy())
	if err != nil {
		return err
	}
	defer cleanup(guard.Close)
	var locked bool
	if err = guard.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return fmt.Errorf("scenario %s is already running", id)
	}
	schema := "lab_" + strings.ReplaceAll(id, "-", "_")
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = guard.Exec(ctx, "DROP SCHEMA IF EXISTS "+quoted+" CASCADE; CREATE SCHEMA "+quoted); err != nil {
		return err
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",pg_catalog"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err = pool.Ping(ctx); err != nil {
		return err
	}
	if _, err = pool.Exec(ctx, `CREATE TABLE products(id bigint PRIMARY KEY, stock integer NOT NULL CHECK(stock>=0));
 CREATE TABLE orders(id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, request_key text UNIQUE, product_id bigint NOT NULL REFERENCES products(id), quantity integer NOT NULL CHECK(quantity>0));
 INSERT INTO products VALUES (1,1),(2,1);`); err != nil {
		return err
	}
	return fn(pool)
}

func Run(ctx context.Context, id string, out io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	scenarios := map[string]func(context.Context, *pgxpool.Pool, io.Writer) error{
		"stock-race": stockRace, "isolation": isolation, "deadlock": deadlock, "idempotency": idempotency,
		"query-plan": queryPlan, "job-queue": jobQueue, "long-transaction": longTransaction, "go-pool": goPool,
	}
	fn, ok := scenarios[id]
	if !ok {
		return fmt.Errorf("unknown scenario %q; choose %s", id, strings.Join(IDs, ", "))
	}
	err := WithSchema(ctx, id, func(p *pgxpool.Pool) error { return fn(ctx, p, out) })
	if err == nil {
		_, err = fmt.Fprintln(out, "PASS "+id)
	}
	return err
}

func waitBlocked(ctx context.Context, p *pgxpool.Pool, pid uint32) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var n int
		if err := p.QueryRow(ctx, "SELECT cardinality(pg_blocking_pids($1))", int64(pid)).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
