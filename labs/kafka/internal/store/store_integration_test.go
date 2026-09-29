//go:build integration

package store_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/config"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/event"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/kafka"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/store"
	labsql "github.com/andreygilmiyarov/go-interview/labs/kafka/sql"
)

type integrationFixture struct {
	ctx      context.Context
	cancel   context.CancelFunc
	database *store.Store
	pool     *pgxpool.Pool
	admin    *pgxpool.Pool
	schema   string
	kafka    config.Config
	topic    string
	group    string
	event    event.Event
	consumer string
}

func newIntegrationFixture(t *testing.T) *integrationFixture {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	cfg, err := config.Load()
	if err != nil {
		cancel()
		t.Fatalf("load integration configuration: %v", err)
	}

	admin, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		cancel()
		t.Fatalf("open integration admin connection: %v", err)
	}
	if err := admin.Ping(ctx); err != nil {
		admin.Close()
		cancel()
		t.Fatalf("ping integration PostgreSQL: %v", err)
	}
	schemaName := "kafka_it_" + uniqueSuffix(t)
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+(pgx.Identifier{schemaName}).Sanitize()); err != nil {
		admin.Close()
		cancel()
		t.Fatalf("create isolated integration schema: %v", err)
	}
	scopedURL, err := databaseURLWithSearchPath(cfg.DatabaseURL, schemaName)
	if err != nil {
		dropSchemaAfterSetupFailure(admin, schemaName)
		admin.Close()
		cancel()
		t.Fatalf("configure isolated integration schema: %v", err)
	}
	database, err := store.Open(ctx, scopedURL)
	if err != nil {
		dropSchemaAfterSetupFailure(admin, schemaName)
		admin.Close()
		cancel()
		t.Fatalf("connect integration PostgreSQL: %v", err)
	}
	pool, err := pgxpool.New(ctx, scopedURL)
	if err != nil {
		database.Close()
		dropSchemaAfterSetupFailure(admin, schemaName)
		admin.Close()
		cancel()
		t.Fatalf("open isolated integration connection: %v", err)
	}
	fixture := &integrationFixture{
		ctx:      ctx,
		cancel:   cancel,
		database: database,
		pool:     pool,
		admin:    admin,
		schema:   schemaName,
		kafka:    cfg,
		consumer: "it-consumer-" + uniqueSuffix(t),
	}
	fixture.event = event.Event{
		EventID:     "it-event-" + uniqueSuffix(t),
		OrderID:     "it-order-" + uniqueSuffix(t),
		Type:        "order.created",
		Version:     1,
		AmountCents: 1285,
	}
	t.Cleanup(func() { fixture.close(t) })

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping isolated integration PostgreSQL: %v", err)
	}
	if err := database.Init(ctx, labsql.Schema); err != nil {
		t.Fatalf("initialize integration schema: %v", err)
	}
	return fixture
}

func dropSchemaAfterSetupFailure(admin *pgxpool.Pool, schema string) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = admin.Exec(cleanupCtx, "DROP SCHEMA "+(pgx.Identifier{schema}).Sanitize()+" CASCADE")
}

func (f *integrationFixture) close(t *testing.T) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	f.database.Close()
	f.pool.Close()
	if f.group != "" {
		if err := kafka.DeleteConsumerGroup(cleanupCtx, f.kafka, f.group); err != nil {
			t.Errorf("delete isolated Kafka consumer group: %v", err)
		}
	}
	if f.topic != "" {
		if err := kafka.DeleteTopic(cleanupCtx, f.kafka, f.topic); err != nil {
			t.Errorf("delete isolated Kafka topic: %v", err)
		}
	}
	if _, err := f.admin.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+(pgx.Identifier{f.schema}).Sanitize()+" CASCADE"); err != nil {
		t.Errorf("drop isolated PostgreSQL schema: %v", err)
	}
	f.admin.Close()
	f.cancel()
}

func databaseURLWithSearchPath(databaseURL, schemaName string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("parse database URL: %w", err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schemaName)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func uniqueSuffix(t *testing.T) string {
	t.Helper()
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		t.Fatalf("generate integration test identity: %v", err)
	}
	return hex.EncodeToString(value[:])
}

func TestCreateOrderRollsBackWhenOutboxInsertFails(t *testing.T) {
	fixture := newIntegrationFixture(t)
	body, err := event.Encode(fixture.event)
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.pool.Exec(fixture.ctx,
		"INSERT INTO outbox (event_id, order_id, payload) VALUES ($1, $2, $3)",
		fixture.event.EventID, fixture.event.OrderID, string(body),
	)
	if err != nil {
		t.Fatalf("seed the conflicting outbox row: %v", err)
	}

	if err := fixture.database.CreateOrder(fixture.ctx, fixture.event); err == nil {
		t.Fatal("CreateOrder() succeeded despite the outbox primary-key conflict")
	}
	var orders int64
	if err := fixture.pool.QueryRow(fixture.ctx,
		"SELECT count(*) FROM orders WHERE order_id = $1", fixture.event.OrderID,
	).Scan(&orders); err != nil {
		t.Fatalf("count orders after failed transaction: %v", err)
	}
	if orders != 0 {
		t.Fatalf("orders after failed transaction = %d, want 0", orders)
	}
}

func TestApplyEventConcurrentDuplicatesChangeProjectionOnce(t *testing.T) {
	fixture := newIntegrationFixture(t)
	const workers = 24
	type result struct {
		applied bool
		err     error
	}
	results := make(chan result, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			applied, err := fixture.database.ApplyEvent(fixture.ctx, fixture.consumer, fixture.event)
			results <- result{applied: applied, err: err}
		}()
	}
	wait.Wait()
	close(results)

	appliedCount := 0
	for outcome := range results {
		if outcome.err != nil {
			t.Errorf("ApplyEvent() error: %v", outcome.err)
			continue
		}
		if outcome.applied {
			appliedCount++
		}
	}
	if appliedCount != 1 {
		t.Fatalf("newly applied events = %d, want 1", appliedCount)
	}
	total, err := fixture.database.OrderTotal(fixture.ctx, fixture.event.OrderID)
	if err != nil {
		t.Fatalf("read order total: %v", err)
	}
	if total != fixture.event.AmountCents {
		t.Fatalf("order total = %d, want %d", total, fixture.event.AmountCents)
	}
	count, err := fixture.database.ProcessedCount(fixture.ctx, fixture.consumer)
	if err != nil {
		t.Fatalf("read processed event count: %v", err)
	}
	if count != 1 {
		t.Fatalf("processed event count = %d, want 1", count)
	}
}

func TestApplyEventRollsBackProcessedMarkerWhenProjectionFails(t *testing.T) {
	fixture := newIntegrationFixture(t)
	const maxInt64 = int64(1<<63 - 1)
	if _, err := fixture.pool.Exec(fixture.ctx,
		"INSERT INTO order_totals (order_id, total_cents) VALUES ($1, $2)",
		fixture.event.OrderID, maxInt64,
	); err != nil {
		t.Fatalf("seed total that overflows on increment: %v", err)
	}

	applied, err := fixture.database.ApplyEvent(fixture.ctx, fixture.consumer, fixture.event)
	if err == nil || applied {
		t.Fatalf("ApplyEvent() = (%t, %v), want a projection overflow error", applied, err)
	}
	count, err := fixture.database.ProcessedCount(fixture.ctx, fixture.consumer)
	if err != nil {
		t.Fatalf("count processed events after projection error: %v", err)
	}
	if count != 0 {
		t.Fatalf("processed marker count after projection error = %d, want 0", count)
	}
	total, err := fixture.database.OrderTotal(fixture.ctx, fixture.event.OrderID)
	if err != nil {
		t.Fatalf("read total after projection error: %v", err)
	}
	if total != maxInt64 {
		t.Fatalf("order total after projection error = %d, want %d", total, maxInt64)
	}
}

func TestOutboxReplayAfterKafkaConfirmationAppliesProjectionOnce(t *testing.T) {
	fixture := newIntegrationFixture(t)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load Kafka configuration: %v", err)
	}
	cfg.ReplicationFactor = 1
	topic := "outbox-it-" + uniqueSuffix(t)
	fixture.topic = topic
	fixture.kafka = cfg
	if err := kafka.CreateTopic(fixture.ctx, cfg, topic, 1, 1); err != nil {
		t.Fatalf("create isolated Kafka topic: %v", err)
	}
	producer, err := kafka.NewProducer(cfg)
	if err != nil {
		t.Fatalf("create Kafka producer: %v", err)
	}
	defer producer.Close()

	if err := fixture.database.CreateOrder(fixture.ctx, fixture.event); err != nil {
		t.Fatalf("create order and outbox event: %v", err)
	}
	var sent []event.Event
	send := func(ctx context.Context, outgoing event.Event) error {
		if _, err := kafka.ProduceEvent(ctx, producer, topic, outgoing); err != nil {
			return err
		}
		sent = append(sent, outgoing)
		return nil
	}

	published, err := fixture.database.PublishPending(fixture.ctx, 1, send, true)
	if !errors.Is(err, store.ErrFailAfterPublishBeforeMark) {
		t.Fatalf("first PublishPending() error = %v, want the premark failpoint", err)
	}
	if published != 0 {
		t.Fatalf("first PublishPending() count = %d, want 0 for rolled-back transaction", published)
	}
	pending, err := fixture.database.PendingCount(fixture.ctx)
	if err != nil {
		t.Fatalf("count pending events after failpoint: %v", err)
	}
	if pending != 1 {
		t.Fatalf("pending events after failpoint = %d, want 1", pending)
	}

	published, err = fixture.database.PublishPending(fixture.ctx, 1, send, false)
	if err != nil {
		t.Fatalf("retry PublishPending(): %v", err)
	}
	if published != 1 || len(sent) != 2 || sent[0] != fixture.event || sent[1] != fixture.event {
		t.Fatalf("publish retry results: count=%d, confirmed sends=%#v", published, sent)
	}
	pending, err = fixture.database.PendingCount(fixture.ctx)
	if err != nil {
		t.Fatalf("count pending events after retry: %v", err)
	}
	if pending != 0 {
		t.Fatalf("pending events after retry = %d, want 0", pending)
	}

	group := "outbox-it-" + uniqueSuffix(t)
	fixture.group = group
	consumer, err := kafka.NewConsumer(cfg, topic, group, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	if err != nil {
		t.Fatalf("create Kafka consumer: %v", err)
	}
	defer consumer.Close()
	seen := make([]event.Event, 0, 2)
	applied := make([]bool, 0, 2)
	for len(seen) < 2 {
		fetches := consumer.PollRecords(fixture.ctx, 2-len(seen))
		if fetchErr := fetches.Err(); fetchErr != nil {
			t.Fatalf("poll Kafka records: %v", fetchErr)
		}
		var recordErr error
		fetches.EachRecord(func(record *kgo.Record) {
			if recordErr != nil {
				return
			}
			received, err := kafka.DecodeRecord(record)
			if err != nil {
				recordErr = fmt.Errorf("decode Kafka record: %w", err)
				return
			}
			wasApplied, err := fixture.database.ApplyEvent(fixture.ctx, fixture.consumer, received)
			if err != nil {
				recordErr = fmt.Errorf("apply consumed event: %w", err)
				return
			}
			if err := consumer.CommitRecords(fixture.ctx, record); err != nil {
				recordErr = fmt.Errorf("commit consumed record after database transaction: %w", err)
				return
			}
			seen = append(seen, received)
			applied = append(applied, wasApplied)
		})
		if recordErr != nil {
			t.Fatal(recordErr)
		}
	}
	if len(seen) != 2 || seen[0] != fixture.event || seen[1] != fixture.event {
		t.Fatalf("consumed replay records = %#v, want the same event twice", seen)
	}
	if len(applied) != 2 || !applied[0] || applied[1] {
		t.Fatalf("database application results = %#v, want [true false]", applied)
	}
	total, err := fixture.database.OrderTotal(fixture.ctx, fixture.event.OrderID)
	if err != nil {
		t.Fatalf("read projected order total: %v", err)
	}
	if total != fixture.event.AmountCents {
		t.Fatalf("projected order total = %d, want %d", total, fixture.event.AmountCents)
	}
}

func TestPublishPendingSendFailureKeepsOutboxEventPending(t *testing.T) {
	fixture := newIntegrationFixture(t)
	if err := fixture.database.CreateOrder(fixture.ctx, fixture.event); err != nil {
		t.Fatalf("create order and outbox event: %v", err)
	}
	sent := 0
	published, err := fixture.database.PublishPending(fixture.ctx, 1, func(context.Context, event.Event) error {
		sent++
		return errors.New("simulated broker error")
	}, false)
	if err == nil {
		t.Fatal("PublishPending() succeeded after the send callback failed")
	}
	if published != 0 || sent != 1 {
		t.Fatalf("publish results: count=%d, send calls=%d", published, sent)
	}
	pending, err := fixture.database.PendingCount(fixture.ctx)
	if err != nil {
		t.Fatalf("count pending events after send failure: %v", err)
	}
	if pending != 1 {
		t.Fatalf("pending events after send failure = %d, want 1", pending)
	}
}
