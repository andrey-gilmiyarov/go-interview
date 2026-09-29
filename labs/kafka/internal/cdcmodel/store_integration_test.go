//go:build integration

package cdcmodel

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultCDCURL = "postgres://cdc_app:cdc_app@127.0.0.1:25432/cdc_lab?sslmode=disable"

type integrationFixture struct {
	ctx    context.Context
	cancel context.CancelFunc
	store  *Store
	pool   *pgxpool.Pool
	admin  *pgxpool.Pool
	schema string
}

func newIntegrationFixture(t *testing.T) *integrationFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	fixture := &integrationFixture{ctx: ctx, cancel: cancel}
	t.Cleanup(func() { fixture.close(t) })

	appURL := os.Getenv("CDC_DATABASE_URL")
	if appURL == "" {
		appURL = defaultCDCURL
	}
	adminURL := os.Getenv("CDC_ADMIN_DATABASE_URL")
	if adminURL == "" {
		t.Fatal("CDC_ADMIN_DATABASE_URL is required so CDC integration tests can use an isolated schema")
	}
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatalf("open CDC integration admin connection: %v", err)
	}
	fixture.admin = admin
	if err := admin.Ping(ctx); err != nil {
		t.Fatalf("ping CDC integration admin connection: %v", err)
	}

	fixture.schema = "cdc_it_" + uniqueSuffix(t)
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+(pgx.Identifier{fixture.schema}).Sanitize()); err != nil {
		t.Fatalf("create isolated CDC integration schema: %v", err)
	}
	appURL = withSearchPath(t, appURL, fixture.schema)
	adminSchemaURL := withSearchPath(t, adminURL, fixture.schema)
	adminSchema, err := pgxpool.New(ctx, adminSchemaURL)
	if err != nil {
		t.Fatalf("open isolated CDC schema connection: %v", err)
	}
	schemaSQL, err := os.ReadFile(filepath.Join("..", "..", "cdc", "schema.sql"))
	if err != nil {
		adminSchema.Close()
		t.Fatalf("read CDC model schema: %v", err)
	}
	if _, err := adminSchema.Exec(ctx, string(schemaSQL), pgx.QueryExecModeSimpleProtocol); err != nil {
		adminSchema.Close()
		t.Fatalf("initialize isolated CDC schema: %v", err)
	}
	adminSchema.Close()

	parsedAppURL, err := url.Parse(appURL)
	if err != nil {
		t.Fatalf("parse CDC application URL: %v", err)
	}
	role := parsedAppURL.User.Username()
	if role == "" {
		t.Fatal("CDC_DATABASE_URL must include an application role")
	}
	schema := (pgx.Identifier{fixture.schema}).Sanitize()
	appRole := (pgx.Identifier{role}).Sanitize()
	if _, err := admin.Exec(ctx, "GRANT USAGE ON SCHEMA "+schema+" TO "+appRole); err != nil {
		t.Fatalf("grant CDC schema usage to application role: %v", err)
	}
	if _, err := admin.Exec(ctx,
		"GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE "+
			schema+".cdc_orders, "+schema+".cdc_commands, "+schema+".cdc_inbox, "+schema+".cdc_projections TO "+appRole,
	); err != nil {
		t.Fatalf("grant CDC state table privileges to application role: %v", err)
	}
	if _, err := admin.Exec(ctx, "GRANT SELECT, INSERT ON TABLE "+schema+".cdc_outbox TO "+appRole); err != nil {
		t.Fatalf("grant CDC outbox privileges to application role: %v", err)
	}

	store, err := Open(ctx, appURL)
	if err != nil {
		t.Fatalf("open CDC model integration store: %v", err)
	}
	fixture.store = store
	pool, err := pgxpool.New(ctx, appURL)
	if err != nil {
		t.Fatalf("open CDC model integration query pool: %v", err)
	}
	fixture.pool = pool
	return fixture
}

func (f *integrationFixture) close(t *testing.T) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if f.store != nil {
		f.store.Close()
	}
	if f.pool != nil {
		f.pool.Close()
	}
	if f.admin != nil {
		if f.schema != "" {
			if _, err := f.admin.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+(pgx.Identifier{f.schema}).Sanitize()+" CASCADE"); err != nil {
				t.Errorf("drop isolated CDC integration schema: %v", err)
			}
		}
		f.admin.Close()
	}
	f.cancel()
}

func withSearchPath(t *testing.T, databaseURL, schema string) string {
	t.Helper()
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("parse CDC database URL: %v", err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func uniqueSuffix(t *testing.T) string {
	t.Helper()
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		t.Fatalf("generate CDC integration identity: %v", err)
	}
	return hex.EncodeToString(value[:])
}

func testCommand(t *testing.T, eventID, orderID string, expectedVersion, amount int64) Command {
	t.Helper()
	typeName := "order.updated"
	if expectedVersion == 0 {
		typeName = "order.created"
	}
	return Command{
		EventID:         eventID,
		OrderID:         orderID,
		Type:            typeName,
		ExpectedVersion: expectedVersion,
		AmountCents:     amount,
	}
}

func testEvent(eventID, orderID, typeName string, orderVersion, amount int64) Event {
	return Event{
		EventID:      eventID,
		OrderID:      orderID,
		Type:         typeName,
		Version:      1,
		OrderVersion: orderVersion,
		AmountCents:  amount,
	}
}

func TestExecuteCreatesAndUpdatesAbsoluteOrderState(t *testing.T) {
	fixture := newIntegrationFixture(t)
	orderID := "order-" + uniqueSuffix(t)

	created, err := fixture.store.Execute(fixture.ctx, testCommand(t, "event-"+uniqueSuffix(t), orderID, 0, 1000))
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if created.Type != "order.created" || created.OrderVersion != 1 || created.AmountCents != 1000 {
		t.Fatalf("create event = %#v", created)
	}
	updated, err := fixture.store.Execute(fixture.ctx, testCommand(t, "event-"+uniqueSuffix(t), orderID, 1, 1500))
	if err != nil {
		t.Fatalf("update order to 1500: %v", err)
	}
	if updated.Type != "order.updated" || updated.OrderVersion != 2 || updated.AmountCents != 1500 {
		t.Fatalf("first update event = %#v", updated)
	}
	updated, err = fixture.store.Execute(fixture.ctx, testCommand(t, "event-"+uniqueSuffix(t), orderID, 2, 1200))
	if err != nil {
		t.Fatalf("update order to 1200: %v", err)
	}
	if updated.OrderVersion != 3 || updated.AmountCents != 1200 {
		t.Fatalf("second update event = %#v", updated)
	}

	state, err := fixture.store.GetOrder(fixture.ctx, orderID)
	if err != nil {
		t.Fatalf("read order state: %v", err)
	}
	if state != (State{OrderID: orderID, OrderVersion: 3, AmountCents: 1200}) {
		t.Fatalf("GetOrder() = %#v, want version 3 and absolute amount 1200", state)
	}
}

func TestExecuteRollsBackOrderAndCommandWhenOutboxInsertFails(t *testing.T) {
	fixture := newIntegrationFixture(t)
	eventID, orderID := "event-"+uniqueSuffix(t), "order-"+uniqueSuffix(t)
	if _, err := fixture.pool.Exec(fixture.ctx,
		"INSERT INTO cdc_outbox (id, aggregatetype, aggregateid, type, payload) VALUES ($1, 'orders', $2, 'order.created', $3::jsonb)",
		eventID, orderID,
		`{"event_id":"`+eventID+`","order_id":"`+orderID+`","type":"order.created","version":1,"order_version":1,"amount_cents":1000}`,
	); err != nil {
		t.Fatalf("seed conflicting outbox row: %v", err)
	}

	if _, err := fixture.store.Execute(fixture.ctx, testCommand(t, eventID, orderID, 0, 1000)); err == nil {
		t.Fatal("Execute() succeeded despite the outbox primary-key conflict")
	}
	for _, query := range []struct {
		name string
		sql  string
		id   string
	}{
		{name: "order", sql: "SELECT count(*) FROM cdc_orders WHERE order_id = $1", id: orderID},
		{name: "command", sql: "SELECT count(*) FROM cdc_commands WHERE event_id = $1", id: eventID},
	} {
		var count int
		if err := fixture.pool.QueryRow(fixture.ctx, query.sql, query.id).Scan(&count); err != nil {
			t.Fatalf("count %s after failed transaction: %v", query.name, err)
		}
		if count != 0 {
			t.Errorf("%s rows after failed transaction = %d, want 0", query.name, count)
		}
	}
}

func TestExecuteConcurrentUpdatesWithSameExpectedVersionHaveOneWinner(t *testing.T) {
	fixture := newIntegrationFixture(t)
	orderID := "order-" + uniqueSuffix(t)
	if _, err := fixture.store.Execute(fixture.ctx, testCommand(t, "event-"+uniqueSuffix(t), orderID, 0, 1000)); err != nil {
		t.Fatalf("create initial order: %v", err)
	}

	const workers = 16
	commands := make([]Command, workers)
	for index := range commands {
		commands[index] = testCommand(t, "event-"+uniqueSuffix(t), orderID, 1, 2000)
	}
	results := make(chan outcomeResult, workers)
	var wait sync.WaitGroup
	for index := range commands {
		wait.Add(1)
		go func(command Command) {
			defer wait.Done()
			created, err := fixture.store.Execute(fixture.ctx, command)
			results <- outcomeResult{event: created, err: err}
		}(commands[index])
	}
	wait.Wait()
	close(results)

	winners, versionErrors := 0, 0
	for result := range results {
		if result.err == nil {
			winners++
		} else if errors.Is(result.err, ErrVersion) {
			versionErrors++
		} else {
			t.Errorf("concurrent Execute() error = %v, want ErrVersion", result.err)
		}
	}
	if winners != 1 || versionErrors != workers-1 {
		t.Fatalf("concurrent outcomes: winners=%d version errors=%d, want 1 and %d", winners, versionErrors, workers-1)
	}
	state, err := fixture.store.GetOrder(fixture.ctx, orderID)
	if err != nil {
		t.Fatalf("read order after concurrent updates: %v", err)
	}
	if state.OrderVersion != 2 || state.AmountCents != 2000 {
		t.Fatalf("order state after concurrent updates = %#v", state)
	}
}

func TestExecuteConcurrentSameCommandIDReturnsOneImmutableResult(t *testing.T) {
	fixture := newIntegrationFixture(t)
	command := testCommand(t, "event-"+uniqueSuffix(t), "order-"+uniqueSuffix(t), 0, 1000)
	const workers = 20
	results := make(chan outcomeResult, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := fixture.store.Execute(fixture.ctx, command)
			results <- outcomeResult{event: result, err: err}
		}()
	}
	wait.Wait()
	close(results)

	var original Event
	for result := range results {
		if result.err != nil {
			t.Errorf("concurrent same-command Execute() error: %v", result.err)
			continue
		}
		if original == (Event{}) {
			original = result.event
		} else if result.event != original {
			t.Errorf("same command returned inconsistent result: %#v and %#v", original, result.event)
		}
	}
	if original.EventID != command.EventID || original.OrderVersion != 1 || original.AmountCents != command.AmountCents {
		t.Fatalf("same-command result = %#v", original)
	}
}

type outcomeResult struct {
	event Event
	err   error
}

func TestExecuteReplayReturnsOriginalResultAfterLaterUpdates(t *testing.T) {
	fixture := newIntegrationFixture(t)
	create := testCommand(t, "event-"+uniqueSuffix(t), "order-"+uniqueSuffix(t), 0, 1000)
	original, err := fixture.store.Execute(fixture.ctx, create)
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if _, err := fixture.store.Execute(fixture.ctx, testCommand(t, "event-"+uniqueSuffix(t), create.OrderID, 1, 1800)); err != nil {
		t.Fatalf("update order: %v", err)
	}
	replayed, err := fixture.store.Execute(fixture.ctx, create)
	if err != nil {
		t.Fatalf("replay create command: %v", err)
	}
	if replayed != original {
		t.Fatalf("replayed result = %#v, want original result %#v", replayed, original)
	}
}

func TestExecuteConflictsWhenCommandIDHasDifferentParameters(t *testing.T) {
	fixture := newIntegrationFixture(t)
	eventID := "event-" + uniqueSuffix(t)
	first := testCommand(t, eventID, "order-"+uniqueSuffix(t), 0, 1000)
	if _, err := fixture.store.Execute(fixture.ctx, first); err != nil {
		t.Fatalf("execute initial command: %v", err)
	}
	changed := first
	changed.AmountCents++
	if _, err := fixture.store.Execute(fixture.ctx, changed); !errors.Is(err, ErrConflict) {
		t.Fatalf("Execute() with reused event ID and changed amount error = %v, want ErrConflict", err)
	}
}

func TestApplyConcurrentDuplicateChangesProjectionOnce(t *testing.T) {
	fixture := newIntegrationFixture(t)
	projection := "projection-" + uniqueSuffix(t)
	event := testEvent("event-"+uniqueSuffix(t), "order-"+uniqueSuffix(t), "order.created", 1, 1250)
	const workers = 20
	type applyOutcome struct {
		applied bool
		err     error
	}
	results := make(chan applyOutcome, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			applied, err := fixture.store.Apply(fixture.ctx, projection, event)
			results <- applyOutcome{applied: applied, err: err}
		}()
	}
	wait.Wait()
	close(results)

	appliedCount := 0
	for result := range results {
		if result.err != nil {
			t.Errorf("Apply() error: %v", result.err)
		} else if result.applied {
			appliedCount++
		}
	}
	if appliedCount != 1 {
		t.Fatalf("newly applied duplicate events = %d, want 1", appliedCount)
	}
	state, err := fixture.store.GetProjection(fixture.ctx, projection, event.OrderID)
	if err != nil {
		t.Fatalf("read projection: %v", err)
	}
	if state != (State{OrderID: event.OrderID, OrderVersion: 1, AmountCents: 1250}) {
		t.Fatalf("projection state = %#v", state)
	}
}

func TestApplyDeduplicatesSameEventAndConflictsOnChangedPayload(t *testing.T) {
	fixture := newIntegrationFixture(t)
	projection := "projection-" + uniqueSuffix(t)
	event := testEvent("event-"+uniqueSuffix(t), "order-"+uniqueSuffix(t), "order.created", 1, 1250)
	if applied, err := fixture.store.Apply(fixture.ctx, projection, event); err != nil || !applied {
		t.Fatalf("first Apply() = (%t, %v), want (true, nil)", applied, err)
	}
	if applied, err := fixture.store.Apply(fixture.ctx, projection, event); err != nil || applied {
		t.Fatalf("duplicate Apply() = (%t, %v), want (false, nil)", applied, err)
	}
	changed := event
	changed.AmountCents++
	if _, err := fixture.store.Apply(fixture.ctx, projection, changed); !errors.Is(err, ErrConflict) {
		t.Fatalf("Apply() with reused event ID and changed payload error = %v, want ErrConflict", err)
	}
	state, err := fixture.store.GetProjection(fixture.ctx, projection, event.OrderID)
	if err != nil {
		t.Fatalf("read projection after conflicting duplicate: %v", err)
	}
	if state.AmountCents != 1250 || state.OrderVersion != 1 {
		t.Fatalf("projection after conflicting duplicate = %#v", state)
	}
}

func TestApplyRejectsVersionGapAndStaleEventWithoutRecordingFailedDelivery(t *testing.T) {
	fixture := newIntegrationFixture(t)
	projection := "projection-" + uniqueSuffix(t)
	orderID := "order-" + uniqueSuffix(t)
	created := testEvent("event-"+uniqueSuffix(t), orderID, "order.created", 1, 1000)
	if applied, err := fixture.store.Apply(fixture.ctx, projection, created); err != nil || !applied {
		t.Fatalf("apply create = (%t, %v), want (true, nil)", applied, err)
	}
	update3 := testEvent("event-"+uniqueSuffix(t), orderID, "order.updated", 3, 1300)
	if _, err := fixture.store.Apply(fixture.ctx, projection, update3); !errors.Is(err, ErrVersion) {
		t.Fatalf("Apply() with version gap error = %v, want ErrVersion", err)
	}
	update2 := testEvent("event-"+uniqueSuffix(t), orderID, "order.updated", 2, 1200)
	if applied, err := fixture.store.Apply(fixture.ctx, projection, update2); err != nil || !applied {
		t.Fatalf("apply version 2 after rejected gap = (%t, %v), want (true, nil)", applied, err)
	}
	if applied, err := fixture.store.Apply(fixture.ctx, projection, update3); err != nil || !applied {
		t.Fatalf("retry version 3 after filling gap = (%t, %v), want (true, nil)", applied, err)
	}
	stale := testEvent("event-"+uniqueSuffix(t), orderID, "order.created", 1, 1000)
	if _, err := fixture.store.Apply(fixture.ctx, projection, stale); !errors.Is(err, ErrVersion) {
		t.Fatalf("Apply() with unknown stale event error = %v, want ErrVersion", err)
	}
}

func TestGetOrderAndProjectionReturnNotFound(t *testing.T) {
	fixture := newIntegrationFixture(t)
	if _, err := fixture.store.GetOrder(fixture.ctx, "missing-"+uniqueSuffix(t)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetOrder() error = %v, want ErrNotFound", err)
	}
	if _, err := fixture.store.GetProjection(fixture.ctx, "projection-"+uniqueSuffix(t), "missing-"+uniqueSuffix(t)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetProjection() error = %v, want ErrNotFound", err)
	}
}

func TestApplyRejectsUpdateBeforeCreate(t *testing.T) {
	fixture := newIntegrationFixture(t)
	event := testEvent("event-"+uniqueSuffix(t), "order-"+uniqueSuffix(t), "order.updated", 2, 1200)
	if _, err := fixture.store.Apply(fixture.ctx, "projection-"+uniqueSuffix(t), event); !errors.Is(err, ErrVersion) {
		t.Fatalf("Apply() update before create error = %v, want ErrVersion", err)
	}
}

func TestExecuteReportsOptimisticVersionMismatch(t *testing.T) {
	fixture := newIntegrationFixture(t)
	orderID := "order-" + uniqueSuffix(t)
	if _, err := fixture.store.Execute(fixture.ctx, testCommand(t, "event-"+uniqueSuffix(t), orderID, 0, 1000)); err != nil {
		t.Fatalf("create order: %v", err)
	}
	if _, err := fixture.store.Execute(fixture.ctx, testCommand(t, "event-"+uniqueSuffix(t), orderID, 0, 1200)); !errors.Is(err, ErrConflict) {
		t.Fatalf("create existing order error = %v, want ErrConflict", err)
	}
	if _, err := fixture.store.Execute(fixture.ctx, testCommand(t, "event-"+uniqueSuffix(t), orderID, 2, 1200)); !errors.Is(err, ErrVersion) {
		t.Fatalf("update with wrong expected version error = %v, want ErrVersion", err)
	}
	if _, err := fixture.store.Execute(fixture.ctx, testCommand(t, "event-"+uniqueSuffix(t), "missing-"+uniqueSuffix(t), 1, 1200)); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update missing order error = %v, want ErrNotFound", err)
	}
	if _, err := fixture.store.Execute(fixture.ctx, testCommand(t, "event-"+uniqueSuffix(t), orderID, int64(^uint64(0)>>1), 1200)); !errors.Is(err, ErrVersion) {
		t.Fatalf("update with maximum expected version error = %v, want ErrVersion", err)
	}
}
