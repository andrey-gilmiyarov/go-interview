//go:build cdcintegration

package cdce2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/cdcconsumer"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/cdcmodel"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/config"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/kafka"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	caseTimeout    = 110 * time.Second
	projectionWait = 90 * time.Second
	composeTimeout = 35 * time.Second
)

type labEnvironment struct {
	mode       string
	database   string
	brokers    []string
	connectURL string
	project    string
	compose    string
	moduleDir  string
}

type fixture struct {
	env   labEnvironment
	ctx   context.Context
	store *cdcmodel.Store
	pool  *pgxpool.Pool
}

func TestOrderEventsReachProjectionAndCommandReplayAddsNoOutboxRow(t *testing.T) {
	caseCtx, cancel := context.WithTimeout(context.Background(), caseTimeout)
	defer cancel()
	env := loadEnvironment(t)
	waitForLabReady(t, caseCtx, env, 30*time.Second)
	database := openFixture(t, caseCtx, env)
	projection := "e2e-" + testID(t)
	registerGroupCleanup(t, env.brokers, "cdc."+projection)
	consumer := newConsumer(t, env, projection)
	defer closeConsumer(t, consumer, "cdc."+projection)

	probeOrder := "order-probe-" + testID(t)
	probe, err := database.store.Execute(database.ctx, command("event-probe-"+testID(t), probeOrder, 0, 1))
	if err != nil {
		t.Fatalf("create controlled CDC probe: %v", err)
	}
	wantProbe := cdcmodel.Event{
		EventID:      probe.EventID,
		OrderID:      probeOrder,
		Type:         "order.created",
		Version:      1,
		OrderVersion: 1,
		AmountCents:  1,
	}
	if probe != wantProbe {
		t.Fatalf("controlled probe event = %#v, want %#v", probe, wantProbe)
	}
	waitForProjection(t, database, consumer, projection, probeOrder, 1)
	probeState, err := database.store.GetProjection(database.ctx, projection, probeOrder)
	if err != nil {
		t.Fatalf("read controlled probe projection: %v", err)
	}
	if probeState != (cdcmodel.State{OrderID: probeOrder, OrderVersion: 1, AmountCents: 1}) {
		t.Fatalf("probe projection = %#v, want one cent at version 1", probeState)
	}

	orderID := "order-e2e-" + testID(t)
	createCommand := command("event-create-"+testID(t), orderID, 0, 1000)
	created, err := database.store.Execute(database.ctx, createCommand)
	if err != nil {
		t.Fatalf("create E2E order: %v", err)
	}
	if created.Type != "order.created" || created.OrderVersion != 1 || created.AmountCents != 1000 {
		t.Fatalf("create event = %#v", created)
	}
	if _, err := database.store.Execute(database.ctx, command("event-update-2-"+testID(t), orderID, 1, 1500)); err != nil {
		t.Fatalf("update E2E order to 1500: %v", err)
	}
	if _, err := database.store.Execute(database.ctx, command("event-update-3-"+testID(t), orderID, 2, 1200)); err != nil {
		t.Fatalf("update E2E order to 1200: %v", err)
	}

	replayed, err := database.store.Execute(database.ctx, createCommand)
	if err != nil {
		t.Fatalf("replay original create command: %v", err)
	}
	if replayed != created {
		t.Fatalf("replayed create result = %#v, want immutable original %#v", replayed, created)
	}
	var outboxRows int64
	if err := database.pool.QueryRow(database.ctx,
		"SELECT count(*) FROM cdc_outbox WHERE id = $1", createCommand.EventID,
	).Scan(&outboxRows); err != nil {
		t.Fatalf("count outbox rows for replayed command: %v", err)
	}
	if outboxRows != 1 {
		t.Fatalf("outbox rows for replayed command = %d, want 1", outboxRows)
	}

	waitForProjection(t, database, consumer, projection, orderID, 3)
	state, err := database.store.GetProjection(database.ctx, projection, orderID)
	if err != nil {
		t.Fatalf("read final E2E projection: %v", err)
	}
	if state != (cdcmodel.State{OrderID: orderID, OrderVersion: 3, AmountCents: 1200}) {
		t.Fatalf("final projection = %#v, want absolute amount 1200 at version 3", state)
	}
}

func TestConnectRestartPublishesOutboxRowsWrittenWhileStopped(t *testing.T) {
	caseCtx, cancel := context.WithTimeout(context.Background(), caseTimeout)
	defer cancel()
	env := loadEnvironment(t)
	waitForLabReady(t, caseCtx, env, 30*time.Second)
	database := openFixture(t, caseCtx, env)
	projection := "connect-restart-" + testID(t)
	group := "cdc." + projection
	registerGroupCleanup(t, env.brokers, group)
	consumer := newConsumer(t, env, projection)
	defer closeConsumer(t, consumer, group)

	restore := stopService(t, caseCtx, env, "connect")
	event := database.storeCommand(t, "event-connect-stop-"+testID(t), "order-connect-stop-"+testID(t), 0, 1700)
	if event.Type != "order.created" || event.OrderVersion != 1 {
		t.Fatalf("create while Connect stopped returned %#v", event)
	}
	if err := restore(); err != nil {
		t.Fatalf("restore Kafka Connect after outbox write: %v", err)
	}
	waitForLabReady(t, caseCtx, env, 45*time.Second)
	waitForProjection(t, database, consumer, projection, event.OrderID, 1)
	assertProjection(t, database, projection, event.OrderID, 1, 1700)
}

func TestBrokerRestartRecoversOutboxDelivery(t *testing.T) {
	caseCtx, cancel := context.WithTimeout(context.Background(), caseTimeout)
	defer cancel()
	env := loadEnvironment(t)
	waitForLabReady(t, caseCtx, env, 30*time.Second)
	database := openFixture(t, caseCtx, env)
	projection := "broker-restart-" + testID(t)
	group := "cdc." + projection
	registerGroupCleanup(t, env.brokers, group)
	consumer := newConsumer(t, env, projection)
	consumerClosed := false
	t.Cleanup(func() {
		if !consumerClosed {
			closeConsumer(t, consumer, group)
		}
	})
	orderID := "order-broker-stop-" + testID(t)
	created := database.storeCommand(t, "event-broker-create-"+testID(t), orderID, 0, 1000)
	if created.Type != "order.created" || created.OrderVersion != 1 {
		t.Fatalf("create before broker stop returned %#v", created)
	}
	waitForProjection(t, database, consumer, projection, orderID, 1)
	assertProjection(t, database, projection, orderID, 1, 1000)
	closeConsumer(t, consumer, group)
	consumerClosed = true

	restore := stopService(t, caseCtx, env, "broker")
	updated := database.storeCommand(t, "event-broker-stop-"+testID(t), orderID, 1, 2300)
	if updated.Type != "order.updated" || updated.OrderVersion != 2 {
		t.Fatalf("update while broker stopped returned %#v", updated)
	}
	if err := restore(); err != nil {
		t.Fatalf("restore broker after outbox write: %v", err)
	}
	if err := waitForBroker(caseCtx, env, 30*time.Second); err != nil {
		t.Fatalf("wait for broker after restart: %v", err)
	}
	if err := recoverFailedConnectTasks(t, caseCtx, env, 25*time.Second); err != nil {
		t.Fatalf("wait for connector recovery: %v", err)
	}
	waitForLabReady(t, caseCtx, env, 30*time.Second)
	// Rejoin the same group after broker recovery and resume from its committed offset.
	consumer = newConsumer(t, env, projection)
	consumerClosed = false
	waitForProjection(t, database, consumer, projection, orderID, 2)
	assertProjection(t, database, projection, orderID, 2, 2300)
}

func loadEnvironment(t *testing.T) labEnvironment {
	t.Helper()
	mode := strings.TrimSpace(os.Getenv("CDC_MODE"))
	var project, composeName string
	switch mode {
	case "kraft":
		project, composeName = "kafka-cdc-kraft", "compose.kraft.yaml"
	case "zk":
		project, composeName = "kafka-cdc-zk", "compose.zk.yaml"
	default:
		t.Fatalf("CDC_MODE must be kraft or zk, got %q", mode)
	}
	moduleDir, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve Kafka module directory: %v", err)
	}
	composePath := filepath.Join(moduleDir, "cdc", composeName)
	if configured := strings.TrimSpace(os.Getenv("CDC_COMPOSE_FILE")); configured != "" && filepath.Clean(configured) != filepath.Clean(composePath) {
		t.Fatalf("CDC_COMPOSE_FILE selects %q; mode %q requires %q", configured, mode, composePath)
	}
	if configured := strings.TrimSpace(os.Getenv("CDC_COMPOSE_PROJECT")); configured != "" && configured != project {
		t.Fatalf("CDC_COMPOSE_PROJECT selects %q; mode %q requires %q", configured, mode, project)
	}
	return labEnvironment{
		mode:       mode,
		database:   requiredEnv(t, "CDC_DATABASE_URL"),
		brokers:    splitBrokers(requiredEnv(t, "CDC_KAFKA_BROKERS")),
		connectURL: strings.TrimRight(requiredEnv(t, "CDC_CONNECT_URL"), "/"),
		project:    project,
		compose:    composePath,
		moduleDir:  moduleDir,
	}
}

func openFixture(t *testing.T, ctx context.Context, env labEnvironment) *fixture {
	t.Helper()
	store, err := cdcmodel.Open(ctx, env.database)
	if err != nil {
		t.Fatalf("open CDC E2E model store: %v", err)
	}
	pool, err := pgxpool.New(ctx, env.database)
	if err != nil {
		store.Close()
		t.Fatalf("open CDC E2E query pool: %v", err)
	}
	f := &fixture{env: env, ctx: ctx, store: store, pool: pool}
	t.Cleanup(func() {
		store.Close()
		pool.Close()
	})
	return f
}

func (f *fixture) storeCommand(t *testing.T, eventID, orderID string, expectedVersion, amount int64) cdcmodel.Event {
	t.Helper()
	eventType := "order.updated"
	if expectedVersion == 0 {
		eventType = "order.created"
	}
	event, err := f.store.Execute(f.ctx, cdcmodel.Command{
		EventID:         eventID,
		OrderID:         orderID,
		Type:            eventType,
		ExpectedVersion: expectedVersion,
		AmountCents:     amount,
	})
	if err != nil {
		t.Fatalf("execute CDC E2E command: %v", err)
	}
	return event
}

func command(eventID, orderID string, expectedVersion, amount int64) cdcmodel.Command {
	eventType := "order.updated"
	if expectedVersion == 0 {
		eventType = "order.created"
	}
	return cdcmodel.Command{
		EventID:         eventID,
		OrderID:         orderID,
		Type:            eventType,
		ExpectedVersion: expectedVersion,
		AmountCents:     amount,
	}
}

func newConsumer(t *testing.T, env labEnvironment, projection string) *kgo.Client {
	t.Helper()
	client, err := kgo.NewClient(
		kgo.SeedBrokers(env.brokers...),
		kgo.ClientID("go-interview-cdc-e2e"),
		kgo.ConsumerGroup("cdc."+projection),
		kgo.ConsumeTopics("cdc.orders"),
		kgo.ConsumeStartOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
		kgo.FetchIsolationLevel(kgo.ReadCommitted()),
		kgo.FetchMaxWait(200*time.Millisecond),
		kgo.BlockRebalanceOnPoll(),
	)
	if err != nil {
		t.Fatalf("create CDC E2E consumer: %v", err)
	}
	return client
}

func closeConsumer(t *testing.T, client *kgo.Client, group string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.LeaveGroupContext(ctx); err != nil {
		t.Logf("leave CDC E2E consumer group %q: %v", group, err)
	}
	client.CloseAllowingRebalance()
}

func waitForProjection(t *testing.T, f *fixture, client *kgo.Client, projection, orderID string, version int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(f.ctx, projectionWait)
	defer cancel()
	for {
		state, err := f.store.GetProjection(ctx, projection, orderID)
		if err == nil && state.OrderVersion >= version {
			return
		}
		if err != nil && !errors.Is(err, cdcmodel.ErrNotFound) {
			t.Fatalf("read CDC E2E projection: %v", err)
		}
		if err := ctx.Err(); err != nil {
			t.Fatalf("projection %q order %q did not reach version %d within %s: %v", projection, orderID, version, projectionWait, err)
		}
		err = cdcconsumer.Consume(ctx, client, f.store, projection, 1, false)
		if err != nil {
			t.Fatalf("consume CDC E2E record: %v", err)
		}
	}
}

func assertProjection(t *testing.T, f *fixture, projection, orderID string, version, amount int64) {
	t.Helper()
	state, err := f.store.GetProjection(f.ctx, projection, orderID)
	if err != nil {
		t.Fatalf("read CDC projection: %v", err)
	}
	want := cdcmodel.State{OrderID: orderID, OrderVersion: version, AmountCents: amount}
	if state != want {
		t.Fatalf("projection state = %#v, want %#v", state, want)
	}
}

func waitForLabReady(t *testing.T, parent context.Context, env labEnvironment, timeout time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", filepath.Join(env.moduleDir, "cdc", "lab.sh"), env.mode, "ready")
	cmd.Dir = env.moduleDir
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Run(); err != nil {
		t.Fatalf("CDC lab readiness check failed: %v\n%s", err, output.String())
	}
}

func stopService(t *testing.T, parent context.Context, env labEnvironment, service string) func() error {
	t.Helper()
	if service != "connect" && service != "broker" {
		t.Fatalf("refusing to stop unexpected CDC service %q", service)
	}
	stopped := false
	start := func(timeout time.Duration) error {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := compose(ctx, env, "start", service); err != nil {
			return err
		}
		stopped = false
		return nil
	}
	t.Cleanup(func() {
		if !stopped {
			return
		}
		if err := start(45 * time.Second); err != nil {
			t.Errorf("restore stopped CDC service %s: %v", service, err)
		}
	})
	ctx, cancel := context.WithTimeout(parent, composeTimeout)
	defer cancel()
	stopped = true
	if err := compose(ctx, env, "stop", service); err != nil {
		t.Fatalf("stop CDC service %s: %v", service, err)
	}
	return func() error { return start(45 * time.Second) }
}

func compose(ctx context.Context, env labEnvironment, action string, services ...string) error {
	if action != "stop" && action != "start" && action != "exec" {
		return fmt.Errorf("refusing unexpected Compose action %q", action)
	}
	args := []string{"compose", "--project-name", env.project, "--file", env.compose, action}
	args = append(args, services...)
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Dir = env.moduleDir
	var output bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose %s %s: %w: %s", action, strings.Join(services, " "), err, strings.TrimSpace(output.String()))
	}
	return nil
}

func waitForBroker(parent context.Context, env labEnvironment, timeout time.Duration) error {
	bootstrap := "broker:29092"
	if env.mode == "zk" {
		bootstrap = "broker:19093"
	}
	deadline := time.Now().Add(timeout)
	if parentDeadline, ok := parent.Deadline(); ok && parentDeadline.Before(deadline) {
		deadline = parentDeadline
	}
	var lastErr error
	for time.Now().Before(deadline) {
		ctx, cancel := context.WithTimeout(parent, 5*time.Second)
		lastErr = compose(ctx, env, "exec", "-T", "broker", "/opt/kafka/bin/kafka-topics.sh", "--bootstrap-server", bootstrap, "--list")
		cancel()
		if lastErr == nil {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("broker did not become ready within %s: %w", timeout, lastErr)
}

func recoverFailedConnectTasks(t *testing.T, parent context.Context, env labEnvironment, timeout time.Duration) error {
	t.Helper()
	deadline := time.Now().Add(timeout)
	if parentDeadline, ok := parent.Deadline(); ok && parentDeadline.Before(deadline) {
		deadline = parentDeadline
	}
	restartedFailedTask := false
	client := &http.Client{Timeout: 3 * time.Second}
	for time.Now().Before(deadline) {
		status, err := connectStatus(parent, client, env.connectURL)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		allRunning, failed := connectorStates(status)
		if allRunning {
			return nil
		}
		if failed && !restartedFailedTask {
			if err := restartOnlyFailedTasks(parent, client, env.connectURL); err != nil {
				return fmt.Errorf("restart failed CDC connector task: %w", err)
			}
			t.Log("Kafka Connect reported a failed task after broker restart; requested restart for failed tasks only")
			restartedFailedTask = true
			continue
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("Kafka Connect connector tasks did not return to RUNNING within %s", timeout)
}

func connectStatus(ctx context.Context, client *http.Client, baseURL string) (map[string]any, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/connectors/cdc-orders/status", nil)
	if err != nil {
		return nil, fmt.Errorf("build Connect status request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request Connect status: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read Connect status response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Connect status returned HTTP %d", response.StatusCode)
	}
	var status map[string]any
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("decode Connect status: %w", err)
	}
	return status, nil
}

func connectorStates(status map[string]any) (allRunning, failed bool) {
	connector, _ := status["connector"].(map[string]any)
	connectorState, _ := connector["state"].(string)
	if connectorState == "FAILED" {
		failed = true
	}
	tasks, _ := status["tasks"].([]any)
	allRunning = connectorState == "RUNNING" && len(tasks) > 0
	for _, raw := range tasks {
		task, _ := raw.(map[string]any)
		state, _ := task["state"].(string)
		if state == "FAILED" {
			failed = true
		}
		if state != "RUNNING" {
			allRunning = false
		}
	}
	return allRunning, failed
}

func restartOnlyFailedTasks(ctx context.Context, client *http.Client, baseURL string) error {
	endpoint := baseURL + "/connectors/cdc-orders/restart?includeTasks=true&onlyFailed=true"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build Connect restart request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("request Connect failed-task restart: %w", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Connect failed-task restart returned HTTP %d", response.StatusCode)
	}
	return nil
}

func registerGroupCleanup(t *testing.T, brokers []string, group string) {
	t.Helper()
	cfg := config.Config{Brokers: brokers, ReplicationFactor: 1}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := kafka.DeleteConsumerGroup(ctx, cfg, group); err != nil {
			t.Errorf("delete CDC E2E consumer group %q: %v", group, err)
		}
	})
}

func requiredEnv(t *testing.T, name string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		t.Fatalf("%s is required; run cdc/verify.sh <kraft|zk>", name)
	}
	return value
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

func testID(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
