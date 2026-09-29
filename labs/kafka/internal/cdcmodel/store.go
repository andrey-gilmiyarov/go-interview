package cdcmodel

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const rollbackTimeout = 5 * time.Second

var (
	ErrConflict = errors.New("CDC command or event conflict")
	ErrVersion  = errors.New("CDC order version conflict")
	ErrNotFound = errors.New("CDC order or projection not found")
)

// Command is an idempotent request to create or set an order's absolute amount.
type Command struct {
	EventID         string
	OrderID         string
	Type            string
	ExpectedVersion int64
	AmountCents     int64
}

// State is the current absolute order state stored by the source or a projection.
type State struct {
	OrderID      string `json:"order_id"`
	OrderVersion int64  `json:"order_version"`
	AmountCents  int64  `json:"amount_cents"`
}

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("PostgreSQL database URL is required")
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

func (s *Store) lock(ctx context.Context, tx pgx.Tx, namespace, value string) error {
	digest := sha256.Sum256([]byte(namespace + "\x00" + value))
	key := int64(binary.BigEndian.Uint64(digest[:8]))
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", key); err != nil {
		return fmt.Errorf("lock CDC %s: %w", namespace, err)
	}
	return nil
}

func begin(ctx context.Context, pool *pgxpool.Pool) (pgx.Tx, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin CDC transaction: %w", err)
	}
	return tx, nil
}

func commandEvent(command Command) (Event, error) {
	if strings.TrimSpace(command.EventID) == "" {
		return Event{}, errors.New("event_id is required")
	}
	if strings.TrimSpace(command.OrderID) == "" {
		return Event{}, errors.New("order_id is required")
	}
	if command.AmountCents <= 0 {
		return Event{}, errors.New("amount_cents must be positive")
	}
	if command.ExpectedVersion < 0 {
		return Event{}, fmt.Errorf("%w: expected version must not be negative", ErrVersion)
	}
	if command.ExpectedVersion == math.MaxInt64 {
		return Event{}, fmt.Errorf("%w: order version cannot be incremented", ErrVersion)
	}

	eventType := "order.updated"
	if command.ExpectedVersion == 0 {
		eventType = "order.created"
	}
	if command.Type != eventType {
		return Event{}, fmt.Errorf("command type must be %s for expected version %d", eventType, command.ExpectedVersion)
	}
	event := Event{
		EventID:      command.EventID,
		OrderID:      command.OrderID,
		Type:         eventType,
		Version:      1,
		OrderVersion: command.ExpectedVersion + 1,
		AmountCents:  command.AmountCents,
	}
	if err := event.Validate(); err != nil {
		return Event{}, fmt.Errorf("validate CDC command: %w", err)
	}
	return event, nil
}

// #region execute
func (s *Store) Execute(ctx context.Context, command Command) (Event, error) {
	event, err := commandEvent(command)
	if err != nil {
		return Event{}, err
	}
	payload, err := Encode(event)
	if err != nil {
		return Event{}, err
	}
	fingerprint := Fingerprint(event)

	tx, err := begin(ctx, s.pool)
	if err != nil {
		return Event{}, err
	}
	defer rollback(tx)

	if err := s.lock(ctx, tx, "event", event.EventID); err != nil {
		return Event{}, err
	}
	var savedFingerprint string
	var savedResult []byte
	err = tx.QueryRow(ctx,
		"SELECT fingerprint, result FROM cdc_commands WHERE event_id = $1",
		event.EventID,
	).Scan(&savedFingerprint, &savedResult)
	switch {
	case err == nil:
		if savedFingerprint != fingerprint {
			return Event{}, fmt.Errorf("%w: event_id %q was already used for another command", ErrConflict, event.EventID)
		}
		savedEvent, err := Decode(savedResult)
		if err != nil {
			return Event{}, fmt.Errorf("decode saved CDC command result: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return Event{}, fmt.Errorf("commit CDC command replay: %w", err)
		}
		return savedEvent, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return Event{}, fmt.Errorf("read CDC command result: %w", err)
	}
	if err := s.lock(ctx, tx, "order", event.OrderID); err != nil {
		return Event{}, err
	}

	var currentVersion int64
	err = tx.QueryRow(ctx,
		"SELECT order_version FROM cdc_orders WHERE order_id = $1 FOR UPDATE",
		event.OrderID,
	).Scan(&currentVersion)
	if command.ExpectedVersion == 0 {
		switch {
		case err == nil:
			return Event{}, fmt.Errorf("%w: order_id %q already exists", ErrConflict, event.OrderID)
		case !errors.Is(err, pgx.ErrNoRows):
			return Event{}, fmt.Errorf("read CDC order: %w", err)
		}
		if _, err := tx.Exec(ctx,
			"INSERT INTO cdc_orders (order_id, amount_cents, order_version) VALUES ($1, $2, 1)",
			event.OrderID, event.AmountCents,
		); err != nil {
			return Event{}, fmt.Errorf("insert CDC order: %w", err)
		}
	} else {
		if errors.Is(err, pgx.ErrNoRows) {
			return Event{}, fmt.Errorf("%w: order_id %q does not exist", ErrNotFound, event.OrderID)
		}
		if err != nil {
			return Event{}, fmt.Errorf("read CDC order: %w", err)
		}
		if currentVersion != command.ExpectedVersion {
			return Event{}, fmt.Errorf("%w: order_id %q has version %d, expected %d", ErrVersion, event.OrderID, currentVersion, command.ExpectedVersion)
		}
		if currentVersion == math.MaxInt64 {
			return Event{}, fmt.Errorf("%w: order version cannot be incremented", ErrVersion)
		}
		tag, err := tx.Exec(ctx,
			"UPDATE cdc_orders SET amount_cents = $2, order_version = $3 WHERE order_id = $1",
			event.OrderID, event.AmountCents, event.OrderVersion,
		)
		if err != nil {
			return Event{}, fmt.Errorf("update CDC order: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return Event{}, fmt.Errorf("update CDC order affected %d rows, want 1", tag.RowsAffected())
		}
	}

	if _, err := tx.Exec(ctx,
		"INSERT INTO cdc_outbox (id, aggregatetype, aggregateid, type, payload) VALUES ($1, 'orders', $2, $3, $4::jsonb)",
		event.EventID, event.OrderID, event.Type, string(payload),
	); err != nil {
		return Event{}, fmt.Errorf("insert CDC outbox event: %w", err)
	}
	if _, err := tx.Exec(ctx,
		"INSERT INTO cdc_commands (event_id, fingerprint, result) VALUES ($1, $2, $3::jsonb)",
		event.EventID, fingerprint, string(payload),
	); err != nil {
		return Event{}, fmt.Errorf("save CDC command result: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Event{}, fmt.Errorf("commit CDC command: %w", err)
	}
	return event, nil
}

// #endregion execute

func (s *Store) GetOrder(ctx context.Context, orderID string) (State, error) {
	if strings.TrimSpace(orderID) == "" {
		return State{}, errors.New("order_id is required")
	}
	state, err := readState(ctx, s.pool,
		"SELECT order_id, order_version, amount_cents FROM cdc_orders WHERE order_id = $1",
		orderID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return State{}, fmt.Errorf("%w: order_id %q", ErrNotFound, orderID)
	}
	if err != nil {
		return State{}, fmt.Errorf("read CDC order state: %w", err)
	}
	return state, nil
}

func readState(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) (State, error) {
	var state State
	err := pool.QueryRow(ctx, query, args...).Scan(&state.OrderID, &state.OrderVersion, &state.AmountCents)
	return state, err
}

// #region apply
func (s *Store) Apply(ctx context.Context, projection string, event Event) (bool, error) {
	if strings.TrimSpace(projection) == "" {
		return false, errors.New("projection is required")
	}
	if err := event.Validate(); err != nil {
		return false, fmt.Errorf("validate CDC event: %w", err)
	}
	fingerprint := Fingerprint(event)

	tx, err := begin(ctx, s.pool)
	if err != nil {
		return false, err
	}
	defer rollback(tx)

	if err := s.lock(ctx, tx, "event", event.EventID); err != nil {
		return false, err
	}
	var savedFingerprint string
	err = tx.QueryRow(ctx,
		"SELECT fingerprint FROM cdc_inbox WHERE projection = $1 AND event_id = $2",
		projection, event.EventID,
	).Scan(&savedFingerprint)
	if err == nil {
		if savedFingerprint != fingerprint {
			return false, fmt.Errorf("%w: event_id %q has a different payload in projection %q", ErrConflict, event.EventID, projection)
		}
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit CDC duplicate check: %w", err)
		}
		return false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("read CDC inbox: %w", err)
	}
	if err := s.lock(ctx, tx, "projection-order", projection+"\x00"+event.OrderID); err != nil {
		return false, err
	}

	var currentVersion int64
	err = tx.QueryRow(ctx,
		"SELECT order_version FROM cdc_projections WHERE projection = $1 AND order_id = $2 FOR UPDATE",
		projection, event.OrderID,
	).Scan(&currentVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		if event.Type != "order.created" || event.OrderVersion != 1 {
			return false, fmt.Errorf("%w: projection %q has no order %q for event version %d", ErrVersion, projection, event.OrderID, event.OrderVersion)
		}
		if _, err := tx.Exec(ctx,
			"INSERT INTO cdc_projections (projection, order_id, amount_cents, order_version) VALUES ($1, $2, $3, 1)",
			projection, event.OrderID, event.AmountCents,
		); err != nil {
			return false, fmt.Errorf("insert CDC projection state: %w", err)
		}
	} else {
		if err != nil {
			return false, fmt.Errorf("read CDC projection state: %w", err)
		}
		if event.Type != "order.updated" || currentVersion == math.MaxInt64 || event.OrderVersion != currentVersion+1 {
			return false, fmt.Errorf("%w: projection %q has version %d, event has version %d", ErrVersion, projection, currentVersion, event.OrderVersion)
		}
		tag, err := tx.Exec(ctx,
			"UPDATE cdc_projections SET amount_cents = $3, order_version = $4 WHERE projection = $1 AND order_id = $2",
			projection, event.OrderID, event.AmountCents, event.OrderVersion,
		)
		if err != nil {
			return false, fmt.Errorf("update CDC projection state: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return false, fmt.Errorf("update CDC projection affected %d rows, want 1", tag.RowsAffected())
		}
	}
	if _, err := tx.Exec(ctx,
		"INSERT INTO cdc_inbox (projection, event_id, fingerprint) VALUES ($1, $2, $3)",
		projection, event.EventID, fingerprint,
	); err != nil {
		return false, fmt.Errorf("insert CDC inbox event: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit CDC projection update: %w", err)
	}
	return true, nil
}

// #endregion apply

func (s *Store) GetProjection(ctx context.Context, projection, orderID string) (State, error) {
	if strings.TrimSpace(projection) == "" {
		return State{}, errors.New("projection is required")
	}
	if strings.TrimSpace(orderID) == "" {
		return State{}, errors.New("order_id is required")
	}
	state, err := readState(ctx, s.pool,
		"SELECT order_id, order_version, amount_cents FROM cdc_projections WHERE projection = $1 AND order_id = $2",
		projection, orderID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return State{}, fmt.Errorf("%w: projection %q order_id %q", ErrNotFound, projection, orderID)
	}
	if err != nil {
		return State{}, fmt.Errorf("read CDC projection state: %w", err)
	}
	return state, nil
}
