//go:build integration

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/config"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/kafka"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/store"
)

func TestOutboxCommandsReplayAfterPremarkAndPrecommitFailures(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	baseConfig, err := config.Load()
	if err != nil {
		t.Fatalf("load integration configuration: %v", err)
	}
	admin, err := pgxpool.New(ctx, baseConfig.DatabaseURL)
	if err != nil {
		t.Fatalf("open PostgreSQL admin pool: %v", err)
	}
	if err := admin.Ping(ctx); err != nil {
		admin.Close()
		t.Fatalf("ping PostgreSQL: %v", err)
	}

	suffix := uniqueSmokeSuffix(t)
	schema := "outbox_smoke_" + suffix
	topic := "outbox-smoke-" + suffix
	group := "outbox-smoke-" + suffix
	eventID := "smoke-event-" + suffix
	orderID := "smoke-order-" + suffix
	binary := filepath.Join(t.TempDir(), "outbox")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build outbox CLI: %v\n%s", err, output)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+(pgx.Identifier{schema}).Sanitize()); err != nil {
		admin.Close()
		t.Fatalf("create isolated smoke schema: %v", err)
	}
	databaseURL, err := smokeURLWithSearchPath(baseConfig.DatabaseURL, schema)
	if err != nil {
		dropSmokeSchema(t, admin, schema)
		admin.Close()
		t.Fatalf("configure isolated smoke schema: %v", err)
	}
	cliEnv := environmentWith(map[string]string{
		"DATABASE_URL":             databaseURL,
		"KAFKA_BROKERS":            strings.Join(baseConfig.Brokers, ","),
		"KAFKA_TOPIC":              topic,
		"KAFKA_GROUP":              group,
		"KAFKA_REPLICATION_FACTOR": "1",
	})

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := kafka.DeleteConsumerGroup(cleanupCtx, baseConfig, group); err != nil {
			t.Errorf("delete isolated smoke group: %v", err)
		}
		if err := kafka.DeleteTopic(cleanupCtx, baseConfig, topic); err != nil {
			t.Errorf("delete isolated smoke topic: %v", err)
		}
		if _, err := admin.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+(pgx.Identifier{schema}).Sanitize()+" CASCADE"); err != nil {
			t.Errorf("drop isolated smoke schema: %v", err)
		}
		admin.Close()
	})

	runCLI := func(wantExitCode int, wantOutput string, args ...string) {
		t.Helper()
		command := exec.CommandContext(ctx, binary, args...)
		command.Env = cliEnv
		output, err := command.CombinedOutput()
		exitCode := 0
		if err != nil {
			var exitError *exec.ExitError
			if !errors.As(err, &exitError) {
				t.Fatalf("run outbox %q: %v\n%s", args[0], err, output)
			}
			exitCode = exitError.ExitCode()
		}
		if exitCode != wantExitCode {
			t.Fatalf("outbox %q exit code = %d, want %d\n%s", args[0], exitCode, wantExitCode, output)
		}
		if wantOutput != "" && !strings.Contains(string(output), wantOutput) {
			t.Fatalf("outbox %q output does not contain %q\n%s", args[0], wantOutput, output)
		}
	}
	runCLI(0, "outbox schema initialized", "init", "-timeout=15s")
	runCLI(0, eventID, "create", "-event-id="+eventID, "-order-id="+orderID, "-amount-cents=1285", "-timeout=15s")
	runCLI(1, "fail after Kafka confirmation", "publish", "-limit=1", "-fail-after-send-before-mark", "-timeout=15s")
	runCLI(0, "published", "publish", "-limit=1", "-timeout=15s")
	runCLI(1, "injected failure after processing", "consume", "-consumer="+group, "-count=1", "-fail-after-process-before-commit", "-timeout=15s")
	runCLI(0, "Kafka consume batch completed", "consume", "-consumer="+group, "-count=2", "-timeout=15s")

	database, err := store.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open isolated smoke store: %v", err)
	}
	defer database.Close()
	total, err := database.OrderTotal(ctx, orderID)
	if err != nil {
		t.Fatalf("read smoke order total: %v", err)
	}
	if total != 1285 {
		t.Fatalf("smoke order total = %d, want 1285", total)
	}
	processed, err := database.ProcessedCount(ctx, group)
	if err != nil {
		t.Fatalf("read smoke processed count: %v", err)
	}
	if processed != 1 {
		t.Fatalf("smoke processed count = %d, want 1", processed)
	}
	pending, err := database.PendingCount(ctx)
	if err != nil {
		t.Fatalf("read smoke outbox pending count: %v", err)
	}
	if pending != 0 {
		t.Fatalf("smoke pending outbox count = %d, want 0", pending)
	}
}

func smokeURLWithSearchPath(databaseURL, schema string) (string, error) {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return "", fmt.Errorf("parse database URL: %w", err)
	}
	query := parsed.Query()
	query.Set("options", "-csearch_path="+schema)
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func dropSmokeSchema(t *testing.T, admin *pgxpool.Pool, schema string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := admin.Exec(ctx, "DROP SCHEMA IF EXISTS "+(pgx.Identifier{schema}).Sanitize()+" CASCADE"); err != nil {
		t.Errorf("drop isolated smoke schema: %v", err)
	}
}

func uniqueSmokeSuffix(t *testing.T) string {
	t.Helper()
	var random [8]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatalf("generate smoke identity: %v", err)
	}
	return hex.EncodeToString(random[:])
}

func environmentWith(overrides map[string]string) []string {
	environment := make([]string, 0, len(os.Environ())+len(overrides))
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if _, replaced := overrides[key]; !replaced {
			environment = append(environment, entry)
		}
	}
	for key, value := range overrides {
		environment = append(environment, key+"="+value)
	}
	return environment
}
