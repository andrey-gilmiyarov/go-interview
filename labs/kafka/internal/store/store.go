package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/event"
)

const (
	sendTimeout     = 10 * time.Second
	rollbackTimeout = 5 * time.Second
	maxPublishBatch = 1000
)

var ErrFailAfterPublishBeforeMark = errors.New("fail after Kafka confirmation and before outbox mark")

type Store struct {
	pool *pgxpool.Pool
}

type pendingEvent struct {
	id    string
	event event.Event
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("database URL is required")
	}
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL configuration: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), rollbackTimeout)
	defer cancel()
	_ = tx.Rollback(ctx)
}

func (s *Store) Init(ctx context.Context, schemaSQL string) error {
	if strings.TrimSpace(schemaSQL) == "" {
		return errors.New("schema SQL is required")
	}
	if _, err := s.pool.Exec(ctx, schemaSQL, pgx.QueryExecModeSimpleProtocol); err != nil {
		return fmt.Errorf("initialize PostgreSQL schema: %w", err)
	}
	return nil
}

// #region create-order
func (s *Store) CreateOrder(ctx context.Context, orderEvent event.Event) error {
	payload, err := event.Encode(orderEvent)
	if err != nil {
		return fmt.Errorf("validate order event: %w", err)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin order transaction: %w", err)
	}
	defer rollback(tx)

	if _, err := tx.Exec(ctx,
		"INSERT INTO orders (order_id, amount_cents) VALUES ($1, $2)",
		orderEvent.OrderID, orderEvent.AmountCents,
	); err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	if _, err := tx.Exec(ctx,
		"INSERT INTO outbox (event_id, order_id, payload) VALUES ($1, $2, $3)",
		orderEvent.EventID, orderEvent.OrderID, string(payload),
	); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit order and outbox transaction: %w", err)
	}
	return nil
}

// #endregion create-order

// #region publish-pending
func (s *Store) PublishPending(
	ctx context.Context,
	limit int,
	send func(context.Context, event.Event) error,
	failAfterSendBeforeMark bool,
) (int, error) {
	if limit < 1 {
		return 0, errors.New("publish limit must be positive")
	}
	if limit > maxPublishBatch {
		return 0, fmt.Errorf("publish limit must not exceed %d", maxPublishBatch)
	}
	if send == nil {
		return 0, errors.New("outbox send callback is required")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return 0, fmt.Errorf("begin outbox publish transaction: %w", err)
	}
	defer rollback(tx)

	rows, err := tx.Query(ctx,
		"SELECT event_id, payload FROM outbox WHERE published_at IS NULL ORDER BY created_at, event_id LIMIT $1 FOR UPDATE",
		limit,
	)
	if err != nil {
		return 0, fmt.Errorf("select pending outbox events: %w", err)
	}
	pending := make([]pendingEvent, 0, limit)
	for rows.Next() {
		var id string
		var payload []byte
		if err := rows.Scan(&id, &payload); err != nil {
			rows.Close()
			return 0, fmt.Errorf("read pending outbox event: %w", err)
		}
		orderEvent, err := event.Decode(payload)
		if err != nil {
			rows.Close()
			return 0, fmt.Errorf("decode outbox event %q: %w", id, err)
		}
		if orderEvent.EventID != id {
			rows.Close()
			return 0, fmt.Errorf("outbox key %q does not match payload event_id %q", id, orderEvent.EventID)
		}
		pending = append(pending, pendingEvent{id: id, event: orderEvent})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate pending outbox events: %w", err)
	}
	rows.Close()

	published := 0
	for index, candidate := range pending {
		sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
		err := send(sendCtx, candidate.event)
		cancel()
		if err != nil {
			return 0, fmt.Errorf("send outbox event %q: %w", candidate.id, err)
		}
		if index == 0 && failAfterSendBeforeMark {
			return 0, fmt.Errorf("outbox publish failpoint: %w", ErrFailAfterPublishBeforeMark)
		}
		tag, err := tx.Exec(ctx,
			"UPDATE outbox SET published_at = clock_timestamp() WHERE event_id = $1 AND published_at IS NULL",
			candidate.id,
		)
		if err != nil {
			return 0, fmt.Errorf("mark outbox event %q as published: %w", candidate.id, err)
		}
		if tag.RowsAffected() != 1 {
			return 0, fmt.Errorf("mark outbox event %q as published: affected %d rows, want 1", candidate.id, tag.RowsAffected())
		}
		published++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit outbox publish transaction: %w", err)
	}
	return published, nil
}

// #endregion publish-pending

// #region apply-event
func (s *Store) ApplyEvent(ctx context.Context, consumer string, orderEvent event.Event) (bool, error) {
	if strings.TrimSpace(consumer) == "" {
		return false, errors.New("consumer ID is required")
	}
	if err := orderEvent.Validate(); err != nil {
		return false, fmt.Errorf("validate consumed event: %w", err)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return false, fmt.Errorf("begin consumer transaction: %w", err)
	}
	defer rollback(tx)

	tag, err := tx.Exec(ctx,
		"INSERT INTO processed_events (consumer_id, event_id) VALUES ($1, $2) ON CONFLICT (consumer_id, event_id) DO NOTHING",
		consumer, orderEvent.EventID,
	)
	if err != nil {
		return false, fmt.Errorf("record processed event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if _, err := tx.Exec(ctx,
		"INSERT INTO order_totals (order_id, total_cents) VALUES ($1, $2) ON CONFLICT (order_id) DO UPDATE SET total_cents = order_totals.total_cents + EXCLUDED.total_cents, updated_at = clock_timestamp()",
		orderEvent.OrderID, orderEvent.AmountCents,
	); err != nil {
		return false, fmt.Errorf("update order total: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit consumed event: %w", err)
	}
	return true, nil
}

// #endregion apply-event

func (s *Store) OrderTotal(ctx context.Context, orderID string) (int64, error) {
	var total int64
	err := s.pool.QueryRow(ctx,
		"SELECT COALESCE((SELECT total_cents FROM order_totals WHERE order_id = $1), 0)",
		orderID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("read order total: %w", err)
	}
	return total, nil
}

func (s *Store) PendingCount(ctx context.Context) (int64, error) {
	var count int64
	if err := s.pool.QueryRow(ctx, "SELECT count(*) FROM outbox WHERE published_at IS NULL").Scan(&count); err != nil {
		return 0, fmt.Errorf("count pending outbox events: %w", err)
	}
	return count, nil
}

func (s *Store) ProcessedCount(ctx context.Context, consumer string) (int64, error) {
	var count int64
	if err := s.pool.QueryRow(ctx,
		"SELECT count(*) FROM processed_events WHERE consumer_id = $1",
		consumer,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count processed events for consumer: %w", err)
	}
	return count, nil
}
