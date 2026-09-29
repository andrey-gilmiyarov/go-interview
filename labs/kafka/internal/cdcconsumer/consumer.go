package cdcconsumer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/cdcmodel"
	"github.com/twmb/franz-go/pkg/kgo"
)

const recordProcessingTimeout = 10 * time.Second

var errFailAfterDatabaseCommit = errors.New("injected failure after PostgreSQL commit and before Kafka offset commit")

// Consume applies one record at a time. PostgreSQL owns the deduplication and
// projection transaction; Kafka is committed only after that transaction
// succeeds. A crash between those commits causes a safe redelivery.
//
// The caller must configure manual commits, ReadCommitted isolation, the
// desired group, and BlockRebalanceOnPoll. The rebalance block is released on
// every path, including decode, database, failpoint, and commit errors.
// Handlers must honor the bounded per-record context; this function cannot
// forcibly interrupt work that ignores cancellation.
// #region consume-cdc
func Consume(ctx context.Context, client *kgo.Client, store *cdcmodel.Store, projection string, count int, failBeforeCommit bool) error {
	if ctx == nil {
		return errors.New("CDC consume context is required")
	}
	if client == nil {
		return errors.New("CDC Kafka consumer is required")
	}
	if store == nil {
		return errors.New("CDC model store is required")
	}
	if strings.TrimSpace(projection) == "" {
		return errors.New("CDC projection is required")
	}
	if count < 1 {
		return errors.New("CDC record count must be positive")
	}

	processed := 0
	for processed < count {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("consume CDC records: %w", err)
		}
		gotRecord, err := func() (bool, error) {
			defer client.AllowRebalance()

			fetches := client.PollRecords(ctx, 1)
			if fetchErrors := fetches.Errors(); len(fetchErrors) != 0 {
				joined := make([]error, 0, len(fetchErrors))
				for _, fetchErr := range fetchErrors {
					joined = append(joined, fmt.Errorf("topic %q partition %d: %w", fetchErr.Topic, fetchErr.Partition, fetchErr.Err))
				}
				return false, fmt.Errorf("poll CDC Kafka records: %w", errors.Join(joined...))
			}

			records := fetches.Records()
			if len(records) == 0 {
				return false, nil
			}
			// PollRecords(1) should never return more than one, but refuse to
			// commit a batch if the client behavior changes unexpectedly.
			if len(records) != 1 {
				return false, fmt.Errorf("poll CDC Kafka records returned %d records, want at most 1", len(records))
			}
			record := records[0]
			return true, consumeRecord(ctx, client, store, projection, record, failBeforeCommit && processed == 0)
		}()
		if err != nil {
			return err
		}
		if gotRecord {
			processed++
		}
	}
	return nil
}

func consumeRecord(parent context.Context, client *kgo.Client, store *cdcmodel.Store, projection string, record *kgo.Record, failBeforeCommit bool) error {
	recordCtx, cancel := context.WithTimeout(parent, recordProcessingTimeout)
	defer cancel()
	location := fmt.Sprintf("topic %q partition %d offset %d", record.Topic, record.Partition, record.Offset)

	event, err := cdcmodel.Decode(record.Value)
	if err != nil {
		wrapped := fmt.Errorf("decode CDC event at %s: %w", location, err)
		slog.ErrorContext(recordCtx, "CDC event rejected", "topic", record.Topic, "partition", record.Partition, "offset", record.Offset, "error", wrapped)
		return wrapped
	}
	if string(record.Key) != event.OrderID {
		wrapped := fmt.Errorf("CDC Kafka key %q does not match event order_id %q at %s", record.Key, event.OrderID, location)
		slog.ErrorContext(recordCtx, "CDC event rejected", "event_id", event.EventID, "topic", record.Topic, "partition", record.Partition, "offset", record.Offset, "error", wrapped)
		return wrapped
	}

	applied, err := store.Apply(recordCtx, projection, event)
	if err != nil {
		wrapped := fmt.Errorf("apply CDC event %q at %s: %w", event.EventID, location, err)
		slog.ErrorContext(recordCtx, "CDC event application failed", "event_id", event.EventID, "topic", record.Topic, "partition", record.Partition, "offset", record.Offset, "error", wrapped)
		return wrapped
	}
	if err := recordCtx.Err(); err != nil {
		return fmt.Errorf("apply CDC event %q at %s completed after its processing deadline: %w", event.EventID, location, err)
	}
	slog.InfoContext(recordCtx, "CDC event applied", "event_id", event.EventID, "order_id", event.OrderID, "projection", projection, "applied", applied, "topic", record.Topic, "partition", record.Partition, "offset", record.Offset)

	if failBeforeCommit {
		return fmt.Errorf("CDC event %q at %s: %w", event.EventID, location, errFailAfterDatabaseCommit)
	}
	if err := client.CommitRecords(recordCtx, record); err != nil {
		wrapped := fmt.Errorf("commit CDC Kafka offset for event %q at %s: %w", event.EventID, location, err)
		slog.ErrorContext(recordCtx, "CDC Kafka offset commit failed", "event_id", event.EventID, "topic", record.Topic, "partition", record.Partition, "offset", record.Offset, "error", wrapped)
		return wrapped
	}
	return nil
}

// #endregion consume-cdc
