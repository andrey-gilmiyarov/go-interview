//go:build integration

package cdcconsumer

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/cdcmodel"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/config"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/kafka"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	integrationDatabaseURL      = "postgres://cdc_app:cdc_app@127.0.0.1:25432/cdc_lab?sslmode=disable"
	integrationAdminDatabaseURL = "postgres://cdc_admin:cdc_admin@127.0.0.1:25432/cdc_lab?sslmode=disable"
	integrationBrokers          = "127.0.0.1:49092"
)

type testEnvironment struct {
	brokers  []string
	appURL   string
	adminURL string
}

type testDatabase struct {
	store  *cdcmodel.Store
	app    *pgxpool.Pool
	admin  *pgxpool.Pool
	schema string
}

func TestRejectedEventsDoNotAdvanceKafkaOffset(t *testing.T) {
	tests := []struct {
		name string
		bad  func(cdcmodel.Event) (key string, value []byte)
	}{
		{
			name: "malformed payload",
			bad: func(cdcmodel.Event) (string, []byte) {
				return "order-malformed", []byte(`{"event_id":`)
			},
		},
		{
			name: "Kafka key mismatch",
			bad: func(e cdcmodel.Event) (string, []byte) {
				body, _ := cdcmodel.Encode(e)
				return "wrong-order-key", body
			},
		},
		{
			name: "missing order version",
			bad: func(e cdcmodel.Event) (string, []byte) {
				e.Type, e.OrderVersion = "order.updated", 3
				body, _ := cdcmodel.Encode(e)
				return e.OrderID, body
			},
		},
		{
			name: "stale order version",
			bad: func(e cdcmodel.Event) (string, []byte) {
				e.Type, e.OrderVersion = "order.created", 1
				body, _ := cdcmodel.Encode(e)
				return e.OrderID, body
			},
		},
		{
			name: "event ID with conflicting payload",
			bad: func(e cdcmodel.Event) (string, []byte) {
				e.AmountCents++
				body, _ := cdcmodel.Encode(e)
				return e.OrderID, body
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := loadTestEnvironment(t)
			database := newTestDatabase(t, env)
			topic := newTestTopic(t, env.brokers)
			projection := "reject-" + testID(t)
			group := "cdc." + projection
			registerTestGroupCleanup(t, env.brokers, group)
			producer := newTestProducer(t, env.brokers)

			seed := testEvent("seed-"+testID(t), "order-"+testID(t), "order.created", 1, 1000)
			publishEvent(t, producer, topic, seed, seed.OrderID)
			if err := consumeWithGroup(t, env.brokers, topic, group, database.store, projection, 1, false); err != nil {
				t.Fatalf("consume committed seed: %v", err)
			}

			badEvent := seed
			if test.name == "event ID with conflicting payload" {
				badEvent.EventID = seed.EventID
			} else {
				badEvent.EventID = "bad-" + testID(t)
			}
			key, body := test.bad(badEvent)
			publishRaw(t, producer, topic, key, body)
			next := testEvent("next-"+testID(t), seed.OrderID, "order.updated", 2, 1500)
			publishEvent(t, producer, topic, next, next.OrderID)

			firstErr := consumeWithGroup(t, env.brokers, topic, group, database.store, projection, 1, false)
			if firstErr == nil {
				t.Fatal("consume succeeded past a rejected event")
			}
			secondErr := consumeWithGroup(t, env.brokers, topic, group, database.store, projection, 1, false)
			if secondErr == nil {
				t.Fatal("retry advanced beyond the rejected event")
			}
			if firstErr.Error() != secondErr.Error() {
				t.Fatalf("rejected record changed after restart: first=%v second=%v", firstErr, secondErr)
			}
			if !strings.Contains(firstErr.Error(), topic) || !strings.Contains(firstErr.Error(), "partition 0") {
				t.Fatalf("rejection lacks Kafka record location: %v", firstErr)
			}
			state := readProjection(t, database.store, projection, seed.OrderID)
			if state.OrderVersion != 1 || state.AmountCents != 1000 {
				t.Fatalf("rejected record changed projection: %#v", state)
			}
			if test.name != "malformed payload" && test.name != "event ID with conflicting payload" {
				if _, exists := readInboxFingerprint(t, database.app, projection, badEvent.EventID); exists {
					t.Fatalf("rejected event %q was inserted into the projection inbox", badEvent.EventID)
				}
			}
			if test.name == "event ID with conflicting payload" {
				fingerprint, exists := readInboxFingerprint(t, database.app, projection, seed.EventID)
				if !exists || fingerprint != cdcmodel.Fingerprint(seed) {
					t.Fatalf("conflicting retry changed saved inbox entry: fingerprint=%q exists=%t", fingerprint, exists)
				}
			}
		})
	}
}

func TestFailAfterDatabaseCommitRedeliveryDoesNotApplyTwice(t *testing.T) {
	env := loadTestEnvironment(t)
	database := newTestDatabase(t, env)
	topic := newTestTopic(t, env.brokers)
	projection := "failpoint-" + testID(t)
	group := "cdc." + projection
	registerTestGroupCleanup(t, env.brokers, group)
	producer := newTestProducer(t, env.brokers)
	created := testEvent("evt-"+testID(t), "order-"+testID(t), "order.created", 1, 1000)
	publishEvent(t, producer, topic, created, created.OrderID)

	err := consumeWithGroup(t, env.brokers, topic, group, database.store, projection, 1, true)
	if err == nil || !strings.Contains(err.Error(), "after PostgreSQL commit") {
		t.Fatalf("failpoint error = %v, want after-DB-before-Kafka-commit sentinel", err)
	}
	state := readProjection(t, database.store, projection, created.OrderID)
	if state.OrderVersion != 1 || state.AmountCents != 1000 {
		t.Fatalf("projection after failpoint = %#v", state)
	}

	if err := consumeWithGroup(t, env.brokers, topic, group, database.store, projection, 1, false); err != nil {
		t.Fatalf("redeliver after database commit: %v", err)
	}
	state = readProjection(t, database.store, projection, created.OrderID)
	if state.OrderVersion != 1 || state.AmountCents != 1000 {
		t.Fatalf("projection after duplicate delivery = %#v, want one applied event", state)
	}
}

func TestReplayBuildsIndependentProjection(t *testing.T) {
	env := loadTestEnvironment(t)
	database := newTestDatabase(t, env)
	topic := newTestTopic(t, env.brokers)
	firstProjection, secondProjection := "replay-a-"+testID(t), "replay-b-"+testID(t)
	registerTestGroupCleanup(t, env.brokers, "cdc."+firstProjection)
	registerTestGroupCleanup(t, env.brokers, "cdc."+secondProjection)
	producer := newTestProducer(t, env.brokers)
	orderID := "order-" + testID(t)
	events := []cdcmodel.Event{
		testEvent("evt-"+testID(t), orderID, "order.created", 1, 1000),
		testEvent("evt-"+testID(t), orderID, "order.updated", 2, 1500),
		testEvent("evt-"+testID(t), orderID, "order.updated", 3, 1200),
	}
	for _, e := range events {
		publishEvent(t, producer, topic, e, orderID)
	}
	if err := consumeWithGroup(t, env.brokers, topic, "cdc."+firstProjection, database.store, firstProjection, len(events), false); err != nil {
		t.Fatalf("build first projection: %v", err)
	}
	firstState := readProjection(t, database.store, firstProjection, orderID)
	if firstState.OrderVersion != 3 || firstState.AmountCents != 1200 {
		t.Fatalf("first projection = %#v, want absolute amount 1200 at version 3", firstState)
	}

	if err := consumeWithGroup(t, env.brokers, topic, "cdc."+secondProjection, database.store, secondProjection, len(events), false); err != nil {
		t.Fatalf("replay into second projection: %v", err)
	}
	secondState := readProjection(t, database.store, secondProjection, orderID)
	if secondState.OrderVersion != 3 || secondState.AmountCents != 1200 {
		t.Fatalf("replayed projection = %#v, want absolute amount 1200 at version 3", secondState)
	}
	firstStateAfterReplay := readProjection(t, database.store, firstProjection, orderID)
	if firstStateAfterReplay != firstState {
		t.Fatalf("replay changed first projection: before=%#v after=%#v", firstState, firstStateAfterReplay)
	}
}

func loadTestEnvironment(t *testing.T) testEnvironment {
	t.Helper()
	return testEnvironment{
		brokers:  splitBrokers(envOr("CDC_KAFKA_BROKERS", integrationBrokers)),
		appURL:   envOr("CDC_DATABASE_URL", integrationDatabaseURL),
		adminURL: envOr("CDC_ADMIN_DATABASE_URL", integrationAdminDatabaseURL),
	}
}

func newTestDatabase(t *testing.T, env testEnvironment) *testDatabase {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, env.adminURL)
	if err != nil {
		t.Fatalf("open CDC integration admin connection: %v", err)
	}
	if err := admin.Ping(ctx); err != nil {
		admin.Close()
		t.Fatalf("ping CDC integration admin database: %v", err)
	}
	schema := "cdc_consumer_it_" + testID(t)
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+(pgx.Identifier{schema}).Sanitize()); err != nil {
		admin.Close()
		t.Fatalf("create isolated consumer test schema: %v", err)
	}
	adminURL := withSearchPath(t, env.adminURL, schema)
	appURL := withSearchPath(t, env.appURL, schema)
	schemaAdmin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		admin.Close()
		t.Fatalf("open isolated consumer test schema: %v", err)
	}
	schemaSQL, err := os.ReadFile(filepath.Join("..", "..", "cdc", "schema.sql"))
	if err != nil {
		schemaAdmin.Close()
		admin.Close()
		t.Fatalf("read CDC schema: %v", err)
	}
	if _, err := schemaAdmin.Exec(ctx, string(schemaSQL), pgx.QueryExecModeSimpleProtocol); err != nil {
		schemaAdmin.Close()
		admin.Close()
		t.Fatalf("initialize isolated consumer test schema: %v", err)
	}
	schemaAdmin.Close()
	appParsed, err := url.Parse(appURL)
	if err != nil {
		admin.Close()
		t.Fatalf("parse CDC application URL: %v", err)
	}
	role := appParsed.User.Username()
	if role == "" {
		admin.Close()
		t.Fatal("CDC_DATABASE_URL must specify an application role")
	}
	if _, err := admin.Exec(ctx, "GRANT USAGE ON SCHEMA "+(pgx.Identifier{schema}).Sanitize()+" TO "+(pgx.Identifier{role}).Sanitize()); err != nil {
		admin.Close()
		t.Fatalf("grant CDC schema usage: %v", err)
	}
	if _, err := admin.Exec(ctx, "GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA "+(pgx.Identifier{schema}).Sanitize()+" TO "+(pgx.Identifier{role}).Sanitize()); err != nil {
		admin.Close()
		t.Fatalf("grant CDC table privileges: %v", err)
	}
	app, err := pgxpool.New(ctx, appURL)
	if err != nil {
		admin.Close()
		t.Fatalf("open isolated CDC application pool: %v", err)
	}
	store, err := cdcmodel.Open(ctx, appURL)
	if err != nil {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _ = admin.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+(pgx.Identifier{schema}).Sanitize()+" CASCADE")
		cleanupCancel()
		app.Close()
		admin.Close()
		t.Fatalf("open isolated CDC store: %v", err)
	}
	fixture := &testDatabase{store: store, app: app, admin: admin, schema: schema}
	t.Cleanup(func() {
		store.Close()
		app.Close()
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+(pgx.Identifier{schema}).Sanitize()+" CASCADE"); err != nil {
			t.Errorf("drop isolated consumer test schema: %v", err)
		}
		admin.Close()
	})
	return fixture
}

func newTestTopic(t *testing.T, brokers []string) string {
	t.Helper()
	topic := "cdc-consumer-it-" + testID(t)
	cfg := config.Config{Brokers: brokers, ReplicationFactor: 1}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	err := kafka.CreateTopic(ctx, cfg, topic, 1, 0)
	cancel()
	if err != nil {
		t.Fatalf("create isolated CDC test topic: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := kafka.DeleteTopic(cleanupCtx, cfg, topic); err != nil {
			t.Errorf("delete isolated CDC test topic %q: %v", topic, err)
		}
	})
	return topic
}

func newTestProducer(t *testing.T, brokers []string) *kgo.Client {
	t.Helper()
	producer, err := kgo.NewClient(kgo.SeedBrokers(brokers...), kgo.RequiredAcks(kgo.AllISRAcks()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(producer.Close)
	return producer
}

func consumeWithGroup(t *testing.T, brokers []string, topic, group string, store *cdcmodel.Store, projection string, count int, failBeforeCommit bool) error {
	t.Helper()
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topic),
		kgo.ConsumeStartOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
		kgo.FetchIsolationLevel(kgo.ReadCommitted()),
		kgo.FetchMaxWait(200*time.Millisecond),
		kgo.BlockRebalanceOnPoll(),
	)
	if err != nil {
		t.Fatalf("create CDC integration consumer: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	err = Consume(ctx, client, store, projection, count, failBeforeCommit)
	cancel()
	leaveCtx, leaveCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if leaveErr := client.LeaveGroupContext(leaveCtx); leaveErr != nil {
		t.Logf("leave CDC integration group %q: %v", group, leaveErr)
	}
	leaveCancel()
	client.CloseAllowingRebalance()
	return err
}

func registerTestGroupCleanup(t *testing.T, brokers []string, group string) {
	t.Helper()
	cfg := config.Config{Brokers: brokers, ReplicationFactor: 1}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := kafka.DeleteConsumerGroup(ctx, cfg, group); err != nil {
			t.Errorf("delete isolated CDC consumer group %q: %v", group, err)
		}
	})
}

func publishEvent(t *testing.T, producer *kgo.Client, topic string, event cdcmodel.Event, key string) {
	t.Helper()
	body, err := cdcmodel.Encode(event)
	if err != nil {
		t.Fatalf("encode CDC integration event: %v", err)
	}
	publishRaw(t, producer, topic, key, body)
}

func publishRaw(t *testing.T, producer *kgo.Client, topic, key string, body []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := producer.ProduceSync(ctx, &kgo.Record{Topic: topic, Key: []byte(key), Value: body}).FirstErr(); err != nil {
		t.Fatalf("produce CDC integration record: %v", err)
	}
}

func testEvent(eventID, orderID, eventType string, orderVersion, amount int64) cdcmodel.Event {
	return cdcmodel.Event{EventID: eventID, OrderID: orderID, Type: eventType, Version: 1, OrderVersion: orderVersion, AmountCents: amount}
}

func testID(t *testing.T) string {
	t.Helper()
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(value[:])
}

func splitBrokers(value string) []string {
	var brokers []string
	for _, broker := range strings.Split(value, ",") {
		if broker = strings.TrimSpace(broker); broker != "" {
			brokers = append(brokers, broker)
		}
	}
	return brokers
}

func withSearchPath(t *testing.T, databaseURL, schema string) string {
	t.Helper()
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func readProjection(t *testing.T, store *cdcmodel.Store, projection, orderID string) cdcmodel.State {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	state, err := store.GetProjection(ctx, projection, orderID)
	if err != nil {
		t.Fatalf("read projection %q order %q: %v", projection, orderID, err)
	}
	return state
}

func readInboxFingerprint(t *testing.T, pool *pgxpool.Pool, projection, eventID string) (string, bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var fingerprint string
	err := pool.QueryRow(ctx,
		"SELECT fingerprint FROM cdc_inbox WHERE projection = $1 AND event_id = $2",
		projection, eventID,
	).Scan(&fingerprint)
	if errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows) {
		return "", false
	}
	if err != nil {
		t.Fatalf("read CDC inbox event %q: %v", eventID, err)
	}
	return fingerprint, true
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
