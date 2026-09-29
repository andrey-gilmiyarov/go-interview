package kafka

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/event"
	"github.com/twmb/franz-go/pkg/kgo"
)

var (
	ErrFailAfterProcessBeforeCommit = errors.New("injected failure after processing and before Kafka commit")
	ErrFailAfterTransactionalOutput = errors.New("injected failure after transactional output")
)

const handlerTimeout = 10 * time.Second
const cleanupTimeout = 5 * time.Second
const maxBatchRecords = 64

// RecordHandler must return promptly when its context is canceled; Go cannot
// forcibly stop a handler that ignores its context.
type RecordHandler func(context.Context, *kgo.Record, event.Event) error

// #region consume-sequential

// ConsumeSequential processes and commits one record at a time. A record is
// committed only after its handler succeeds, so a failed handler is redelivered.
func ConsumeSequential(ctx context.Context, client *kgo.Client, count int, failAfterProcessBeforeCommit bool, handler RecordHandler) error {
	if client == nil {
		return fmt.Errorf("Kafka consumer is required")
	}
	if count < 0 {
		return fmt.Errorf("record count cannot be negative")
	}
	if handler == nil && count > 0 {
		return fmt.Errorf("record handler is required")
	}

	for processed := 0; processed < count; {
		if err := ctx.Err(); err != nil {
			return err
		}
		fetches := client.PollRecords(ctx, 1)
		if err := fetchError(fetches); err != nil {
			return err
		}
		records := fetches.Records()
		if len(records) == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
			continue
		}

		record := records[0]
		decoded, err := DecodeRecord(record)
		if err != nil {
			return err
		}
		handlerCtx, cancel := context.WithTimeout(ctx, handlerTimeout)
		err = handler(handlerCtx, record, decoded)
		handlerCtxErr := handlerCtx.Err()
		cancel()
		if err != nil {
			return recordError("process", record, err)
		}
		if handlerCtxErr != nil {
			return recordError("process", record, handlerCtxErr)
		}
		if err := ctx.Err(); err != nil {
			return recordError("process", record, err)
		}
		if failAfterProcessBeforeCommit {
			return recordError("injected failure before commit", record, ErrFailAfterProcessBeforeCommit)
		}
		if err := client.CommitRecords(ctx, record); err != nil {
			return recordError("commit", record, err)
		}
		processed++
	}
	return nil
}

// #endregion consume-sequential

// #region consume-parallel

// ConsumeParallel processes each partition in its own worker while preserving
// record order within that partition. The client must be configured with
// kgo.BlockRebalanceOnPoll; all workers finish before any offsets are committed.
func ConsumeParallel(ctx context.Context, client *kgo.Client, count int, failAfterProcessBeforeCommit bool, handler RecordHandler) error {
	if client == nil {
		return fmt.Errorf("Kafka consumer is required")
	}
	if count < 0 {
		return fmt.Errorf("record count cannot be negative")
	}
	if handler == nil && count > 0 {
		return fmt.Errorf("record handler is required")
	}

	for processed := 0; processed < count; {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := func() error {
			// BlockRebalanceOnPoll fences this entire processing-and-commit section.
			// Release on every path, including fetch and handler failures.
			defer client.AllowRebalance()
			fetches := client.PollRecords(ctx, min(count-processed, maxBatchRecords))
			if err := fetchError(fetches); err != nil {
				return err
			}
			records := fetches.Records()
			if len(records) == 0 {
				return ctx.Err()
			}

			groups := make(map[partitionKey][]*kgo.Record)
			for _, record := range records {
				key := partitionKey{topic: record.Topic, partition: record.Partition}
				groups[key] = append(groups[key], record)
			}
			keys := make([]partitionKey, 0, len(groups))
			for key := range groups {
				keys = append(keys, key)
			}
			sort.Slice(keys, func(i, j int) bool {
				if keys[i].topic == keys[j].topic {
					return keys[i].partition < keys[j].partition
				}
				return keys[i].topic < keys[j].topic
			})

			batchCtx, cancel := context.WithTimeout(ctx, handlerTimeout)
			defer cancel()
			results := make(chan partitionResult, len(keys))
			for _, key := range keys {
				key := key
				partitionRecords := groups[key]
				go func() {
					result := processPartition(batchCtx, key, partitionRecords, handler)
					if result.err != nil {
						cancel()
					}
					results <- result
				}()
			}

			completed := make([]partitionResult, 0, len(keys))
			for range keys {
				completed = append(completed, <-results)
			}
			if err := firstPartitionError(completed, ctx.Err()); err != nil {
				return err
			}
			if failAfterProcessBeforeCommit {
				return recordError("injected failure before commit", completed[0].last, ErrFailAfterProcessBeforeCommit)
			}
			if err := ctx.Err(); err != nil {
				return err
			}

			lastRecords := make([]*kgo.Record, 0, len(completed))
			for _, result := range completed {
				lastRecords = append(lastRecords, result.last)
			}
			if err := client.CommitRecords(batchCtx, lastRecords...); err != nil {
				return fmt.Errorf("commit parallel Kafka batch: %w", err)
			}
			processed += len(records)
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

type partitionKey struct {
	topic     string
	partition int32
}

type partitionResult struct {
	last  *kgo.Record
	count int
	err   error
}

func processPartition(ctx context.Context, key partitionKey, records []*kgo.Record, handler RecordHandler) partitionResult {
	result := partitionResult{}
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			result.err = recordError("process", record, err)
			return result
		}
		decoded, err := DecodeRecord(record)
		if err != nil {
			result.err = err
			return result
		}
		handlerCtx, cancel := context.WithTimeout(ctx, handlerTimeout)
		err = handler(handlerCtx, record, decoded)
		handlerCtxErr := handlerCtx.Err()
		cancel()
		if err != nil {
			result.err = recordError("process", record, err)
			return result
		}
		if handlerCtxErr != nil {
			result.err = recordError("process", record, handlerCtxErr)
			return result
		}
		result.last = record
		result.count++
	}
	if result.count != len(records) {
		result.err = fmt.Errorf("partition %q/%d processed %d of %d records", key.topic, key.partition, result.count, len(records))
	}
	return result
}

func firstPartitionError(results []partitionResult, contextErr error) error {
	var firstCanceled error
	for _, result := range results {
		if result.err == nil {
			continue
		}
		if !errors.Is(result.err, context.Canceled) {
			return result.err
		}
		if firstCanceled == nil {
			firstCanceled = result.err
		}
	}
	if contextErr != nil {
		return contextErr
	}
	return firstCanceled
}

// #endregion consume-parallel

// #region consume-transactionally

// ConsumeTransactionally copies a bounded batch to outputTopic and ends one
// Kafka transaction with both the output records and input offsets committed.
// A false committed result means the session fenced this batch during a group
// rebalance; it is retried without advancing the requested count.
func ConsumeTransactionally(ctx context.Context, session *kgo.GroupTransactSession, outputTopic string, count int, failAfterOutput bool) error {
	if session == nil {
		return fmt.Errorf("transactional Kafka session is required")
	}
	if outputTopic == "" {
		return fmt.Errorf("transactional output topic is required")
	}
	if count < 0 {
		return fmt.Errorf("record count cannot be negative")
	}

	for processed := 0; processed < count; {
		if err := ctx.Err(); err != nil {
			return err
		}
		fetches := session.PollRecords(ctx, min(count-processed, maxBatchRecords))
		if err := fetchError(fetches); err != nil {
			return err
		}
		records := fetches.Records()
		if len(records) == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
			continue
		}

		events := make([]event.Event, 0, len(records))
		outputs := make([]*kgo.Record, 0, len(records))
		for _, record := range records {
			decoded, err := DecodeRecord(record)
			if err != nil {
				return err
			}
			value, err := event.Encode(decoded)
			if err != nil {
				return recordError("encode transactional output", record, err)
			}
			events = append(events, decoded)
			outputs = append(outputs, &kgo.Record{Topic: outputTopic, Key: []byte(decoded.OrderID), Value: value})
		}
		if err := session.Begin(); err != nil {
			return fmt.Errorf("begin Kafka transaction: %w", err)
		}

		transactionCtx, cancel := context.WithTimeout(ctx, handlerTimeout)
		if err := produceTransactionalBatch(transactionCtx, session, outputTopic, events, outputs); err != nil {
			cancel()
			return abortTransaction(session, err)
		}
		if failAfterOutput {
			cancel()
			return abortTransaction(session, ErrFailAfterTransactionalOutput)
		}

		committed, err := session.End(transactionCtx, kgo.TryCommit)
		cancel()
		if err != nil {
			cause := fmt.Errorf("end Kafka transaction: %w", err)
			return abortTransaction(session, cause)
		}
		if !committed {
			continue
		}
		processed += len(records)
	}
	return nil
}

func produceTransactionalBatch(ctx context.Context, session *kgo.GroupTransactSession, outputTopic string, events []event.Event, records []*kgo.Record) error {
	if len(records) != len(events) {
		return fmt.Errorf("transactional output batch has %d records for %d events", len(records), len(events))
	}
	produceCtx, cancel := context.WithTimeout(ctx, handlerTimeout)
	defer cancel()
	type result struct {
		eventID string
		err     error
	}
	results := make(chan result, len(records))
	for i, record := range records {
		eventID := events[i].EventID
		session.Produce(produceCtx, record, func(_ *kgo.Record, err error) {
			results <- result{eventID: eventID, err: err}
		})
	}

	var firstErr error
	remaining := len(records)
	for remaining > 0 {
		select {
		case delivery := <-results:
			remaining--
			if delivery.err != nil && firstErr == nil {
				firstErr = fmt.Errorf("produce transactional event %q to topic %q: %w", delivery.eventID, outputTopic, delivery.err)
			}
		case <-produceCtx.Done():
			return fmt.Errorf("wait for transactional output deliveries: %w", produceCtx.Err())
		}
	}
	return firstErr
}

func abortTransaction(session *kgo.GroupTransactSession, cause error) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()
	committed, err := session.End(cleanupCtx, kgo.TryAbort)
	if err != nil {
		return errors.Join(cause, fmt.Errorf("abort Kafka transaction: %w", err))
	}
	if committed {
		return errors.Join(cause, fmt.Errorf("Kafka transaction unexpectedly committed while aborting"))
	}
	return cause
}

// #endregion consume-transactionally

func fetchError(fetches kgo.Fetches) error {
	var errs []error
	for _, fetchErr := range fetches.Errors() {
		errs = append(errs, fmt.Errorf("fetch topic %q partition %d: %w", fetchErr.Topic, fetchErr.Partition, fetchErr.Err))
	}
	return errors.Join(errs...)
}

func recordError(action string, record *kgo.Record, err error) error {
	return fmt.Errorf("%s topic %q partition %d offset %d: %w", action, record.Topic, record.Partition, record.Offset, err)
}
