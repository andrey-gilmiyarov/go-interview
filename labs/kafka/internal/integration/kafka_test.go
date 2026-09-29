//go:build integration

package integration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/config"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/event"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/kafka"
	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

func testConfig(t *testing.T) config.Config {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func newTopic(t *testing.T, cfg config.Config, partitions int32) string {
	t.Helper()
	topic := uniqueID("kafka-lab-it")
	minISR := int16(0)
	if cfg.ReplicationFactor > 1 {
		minISR = 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	err := kafka.CreateTopic(ctx, cfg, topic, partitions, minISR)
	cancel()
	if err != nil {
		t.Fatalf("create test topic %q: %v", topic, err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := kafka.DeleteTopic(ctx, cfg, topic); err != nil {
			t.Logf("delete owned test topic %q: %v", topic, err)
		}
	})
	return topic
}

func newGroup(t *testing.T, cfg config.Config) string {
	t.Helper()
	group := uniqueID("kafka-lab-it-group")
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := kafka.DeleteConsumerGroup(ctx, cfg, group); err != nil {
			t.Logf("delete owned test group %q: %v", group, err)
		}
	})
	return group
}

func uniqueID(prefix string) string {
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		panic(err)
	}
	return prefix + "-" + hex.EncodeToString(suffix[:])
}

func orderEvent(id string, amount int64) event.Event {
	return event.Event{
		EventID:     id,
		OrderID:     "order-" + id,
		Type:        "order.created",
		Version:     1,
		AmountCents: amount,
	}
}

func produceEvent(t *testing.T, cfg config.Config, client *kgo.Client, topic string, e event.Event) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := kafka.ProduceEvent(ctx, client, topic, e); err != nil {
		t.Fatalf("produce test event %q: %v", e.EventID, err)
	}
}

func pollOne(ctx context.Context, client *kgo.Client) (*kgo.Record, error) {
	for {
		fetches := client.PollRecords(ctx, 1)
		if records := fetches.Records(); len(records) > 0 {
			return records[0], nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			return nil, fmt.Errorf("poll Kafka: %v", errs)
		}
	}
}

func TestDeleteTopicRemovesMetadataWithNegotiatedRequestVersion(t *testing.T) {
	cfg := testConfig(t)
	topic := uniqueID("kafka-lab-delete")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := kafka.CreateTopic(ctx, cfg, topic, 1, 0); err != nil {
		t.Fatalf("create deletion test topic: %v", err)
	}
	deleted := false
	defer func() {
		if !deleted {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cleanupCancel()
			_ = kafka.DeleteTopic(cleanupCtx, cfg, topic)
		}
	}()
	if err := kafka.DeleteTopic(ctx, cfg, topic); err != nil {
		t.Fatalf("delete topic: %v", err)
	}
	deleted = true

	client, err := kgo.NewClient(kgo.SeedBrokers(cfg.Brokers...))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	request := kmsg.NewPtrMetadataRequest()
	request.AllowAutoTopicCreation = false
	request.Topics = []kmsg.MetadataRequestTopic{{Topic: &topic}}
	for {
		response, err := request.RequestWith(ctx, client)
		if err != nil {
			t.Fatalf("request topic metadata after delete: %v", err)
		}
		found := false
		gone := false
		for _, result := range response.Topics {
			if result.Topic == nil || *result.Topic != topic {
				continue
			}
			found = true
			gone = errors.Is(kerr.ErrorForCode(result.ErrorCode), kerr.UnknownTopicOrPartition)
		}
		if gone || !found {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("topic %q still appears in metadata after delete", topic)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func TestThreeBrokerTopicUsesReplicationAndMinISR(t *testing.T) {
	cfg := testConfig(t)
	if cfg.ReplicationFactor != 3 {
		t.Skip("set KAFKA_REPLICATION_FACTOR=3 when targeting the three-broker cluster")
	}
	topic := uniqueID("kafka-lab-rf3")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := kafka.CreateTopic(ctx, cfg, topic, 3, 2); err != nil {
		t.Fatalf("create replication-factor-3 topic with minISR=2: %v", err)
	}
	deleted := false
	defer func() {
		if !deleted {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cleanupCancel()
			_ = kafka.DeleteTopic(cleanupCtx, cfg, topic)
		}
	}()

	admin, err := kgo.NewClient(kgo.SeedBrokers(cfg.Brokers...))
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	request := kmsg.NewPtrMetadataRequest()
	request.AllowAutoTopicCreation = false
	request.Topics = []kmsg.MetadataRequestTopic{{Topic: &topic}}
	response, err := request.RequestWith(ctx, admin)
	if err != nil {
		t.Fatalf("read metadata for RF3 topic: %v", err)
	}
	if len(response.Topics) != 1 || response.Topics[0].Topic == nil || *response.Topics[0].Topic != topic {
		t.Fatalf("metadata response does not identify topic %q: %#v", topic, response.Topics)
	}
	partitions := response.Topics[0].Partitions
	if len(partitions) != 3 {
		t.Fatalf("topic partition count = %d, want 3", len(partitions))
	}
	for _, partition := range partitions {
		if len(partition.Replicas) != 3 {
			t.Fatalf("partition %d has %d replicas, want 3", partition.Partition, len(partition.Replicas))
		}
		if len(partition.ISR) < 2 {
			t.Fatalf("partition %d has %d in-sync replicas, want at least 2", partition.Partition, len(partition.ISR))
		}
	}

	producer, err := kafka.NewProducer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer producer.Close()
	e := orderEvent(uniqueID("evt"), 2750)
	if _, err := kafka.ProduceEvent(ctx, producer, topic, e); err != nil {
		t.Fatalf("produce with all in-sync replicas: %v", err)
	}
	if err := kafka.DeleteTopic(ctx, cfg, topic); err != nil {
		t.Fatalf("delete RF3 test topic: %v", err)
	}
	deleted = true
}

func TestSequentialProcessingFailureBeforeCommitRedelivers(t *testing.T) {
	cfg := testConfig(t)
	topic := newTopic(t, cfg, 1)
	group := newGroup(t, cfg)
	producer, err := kafka.NewProducer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(producer.Close)
	want := orderEvent(uniqueID("evt"), 1250)
	produceEvent(t, cfg, producer, topic, want)

	first, err := kafka.NewConsumer(cfg, topic, group)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(first.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	err = kafka.ConsumeSequential(ctx, first, 1, true, func(_ context.Context, _ *kgo.Record, got event.Event) error {
		if got.EventID != want.EventID {
			return fmt.Errorf("first delivery event ID = %q, want %q", got.EventID, want.EventID)
		}
		return nil
	})
	cancel()
	if !errors.Is(err, kafka.ErrFailAfterProcessBeforeCommit) {
		t.Fatalf("first consume error = %v, want failpoint", err)
	}
	first.Close()

	second, err := kafka.NewConsumer(cfg, topic, group)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(second.Close)
	var redelivered event.Event
	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	err = kafka.ConsumeSequential(ctx, second, 1, false, func(_ context.Context, _ *kgo.Record, got event.Event) error {
		redelivered = got
		return nil
	})
	cancel()
	if err != nil {
		t.Fatal(err)
	}
	if redelivered.EventID != want.EventID {
		t.Fatalf("redelivered event ID = %q, want %q", redelivered.EventID, want.EventID)
	}
}

func TestParallelFailureOnEarlierRecordDoesNotCommitLaterRecord(t *testing.T) {
	cfg := testConfig(t)
	topic := newTopic(t, cfg, 1)
	group := newGroup(t, cfg)
	producer, err := kafka.NewProducer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer producer.Close()

	first := orderEvent(uniqueID("evt"), 1250)
	second := orderEvent(uniqueID("evt"), 1750)
	produceEvent(t, cfg, producer, topic, first)
	produceEvent(t, cfg, producer, topic, second)

	consumer, err := kafka.NewConsumer(cfg, topic, group, kgo.BlockRebalanceOnPoll())
	if err != nil {
		t.Fatal(err)
	}
	consumerClosed := false
	defer func() {
		if !consumerClosed {
			consumer.CloseAllowingRebalance()
		}
	}()

	enteredFirst := make(chan struct{})
	enteredSecond := make(chan struct{}, 1)
	releaseFirst := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	done := make(chan error, 1)
	go func() {
		done <- kafka.ConsumeParallel(ctx, consumer, 2, false, func(handlerCtx context.Context, _ *kgo.Record, got event.Event) error {
			switch got.EventID {
			case first.EventID:
				close(enteredFirst)
				select {
				case <-releaseFirst:
					return fmt.Errorf("injected earlier record failure")
				case <-handlerCtx.Done():
					return handlerCtx.Err()
				}
			case second.EventID:
				enteredSecond <- struct{}{}
				return nil
			default:
				return fmt.Errorf("unexpected event %q", got.EventID)
			}
		})
	}()
	select {
	case <-enteredFirst:
	case <-ctx.Done():
		close(releaseFirst)
		cancel()
		t.Fatalf("first record handler did not start: %v", ctx.Err())
	}
	select {
	case <-enteredSecond:
		close(releaseFirst)
		cancel()
		t.Fatal("later same-partition record started before the earlier record finished")
	case <-time.After(200 * time.Millisecond):
	}
	close(releaseFirst)
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "injected earlier record failure") {
			cancel()
			t.Fatalf("parallel runner error = %v, want earlier record failure", err)
		}
	case <-ctx.Done():
		cancel()
		t.Fatalf("parallel runner did not stop after earlier failure: %v", ctx.Err())
	}
	consumer.CloseAllowingRebalance()
	consumerClosed = true
	cancel()

	retry, err := kafka.NewConsumer(cfg, topic, group)
	if err != nil {
		t.Fatal(err)
	}
	defer retry.CloseAllowingRebalance()
	var redelivered []string
	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	err = kafka.ConsumeSequential(ctx, retry, 2, false, func(_ context.Context, _ *kgo.Record, got event.Event) error {
		redelivered = append(redelivered, got.EventID)
		return nil
	})
	cancel()
	if err != nil {
		t.Fatal(err)
	}
	if len(redelivered) != 2 || redelivered[0] != first.EventID || redelivered[1] != second.EventID {
		t.Fatalf("redelivered IDs = %v, want [%s %s]", redelivered, first.EventID, second.EventID)
	}
}

func TestParallelConsumerBlocksRebalanceUntilPartitionBatchCommits(t *testing.T) {
	cfg := testConfig(t)
	topic := newTopic(t, cfg, 1)
	group := newGroup(t, cfg)
	producer, err := kafka.NewProducer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer producer.Close()

	wanted := orderEvent(uniqueID("evt"), 1250)

	firstAssigned := make(chan int, 4)
	rebalanceBlocked := make(chan struct{}, 4)
	first, err := kafka.NewConsumer(cfg, topic, group,
		kgo.BlockRebalanceOnPoll(),
		kgo.OnPartitionsAssigned(func(_ context.Context, _ *kgo.Client, assigned map[string][]int32) {
			firstAssigned <- len(assigned[topic])
		}),
		kgo.OnPartitionsRevoked(func(_ context.Context, _ *kgo.Client, _ map[string][]int32) {}),
		kgo.OnPartitionsCallbackBlocked(func(_ context.Context, _ *kgo.Client) {
			select {
			case rebalanceBlocked <- struct{}{}:
			default:
			}
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	firstClosed := false
	defer func() {
		if !firstClosed {
			first.CloseAllowingRebalance()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	firstPoll := make(chan error, 1)
	initialPollCtx, stopInitialPoll := context.WithTimeout(ctx, 10*time.Second)
	go func() {
		for {
			fetches := first.PollRecords(initialPollCtx, 1)
			if len(fetches.Records()) > 0 {
				firstPoll <- fmt.Errorf("received a record before the test produced any")
				return
			}
			if initialPollCtx.Err() != nil {
				firstPoll <- initialPollCtx.Err()
				return
			}
			if errs := fetches.Errors(); len(errs) > 0 {
				firstPoll <- fmt.Errorf("initial assignment poll: %v", errs)
				return
			}
		}
	}()
	select {
	case count := <-firstAssigned:
		if count != 1 {
			stopInitialPoll()
			cancel()
			t.Fatalf("initial assigned partitions = %d, want 1", count)
		}
	case <-ctx.Done():
		stopInitialPoll()
		cancel()
		t.Fatalf("wait for initial partition assignment: %v", ctx.Err())
	}
	stopInitialPoll()
	select {
	case err := <-firstPoll:
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			cancel()
			t.Fatalf("initial assignment poll: %v", err)
		}
	case <-ctx.Done():
		cancel()
		t.Fatalf("initial assignment poll did not stop: %v", ctx.Err())
	}

	value, err := event.Encode(wanted)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	record := &kgo.Record{Topic: topic, Partition: 0, Key: []byte(wanted.OrderID), Value: value}
	produceCtx, stopProduce := context.WithTimeout(ctx, 10*time.Second)
	if err := producer.ProduceSync(produceCtx, record).FirstErr(); err != nil {
		stopProduce()
		cancel()
		t.Fatalf("produce partition 0 test event: %v", err)
	}
	stopProduce()

	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	runnerDone := make(chan error, 1)
	go func() {
		runnerDone <- kafka.ConsumeParallel(ctx, first, 1, false, func(handlerCtx context.Context, _ *kgo.Record, got event.Event) error {
			if got.EventID != wanted.EventID {
				return fmt.Errorf("event = %q, want %q", got.EventID, wanted.EventID)
			}
			entered <- struct{}{}
			select {
			case <-release:
				return nil
			case <-handlerCtx.Done():
				return handlerCtx.Err()
			}
		})
	}()
	select {
	case <-entered:
	case <-ctx.Done():
		close(release)
		cancel()
		t.Fatalf("parallel worker did not start: %v", ctx.Err())
	}

	secondAssigned := make(chan int, 4)
	second, err := kafka.NewConsumer(cfg, topic, group,
		kgo.OnPartitionsAssigned(func(_ context.Context, _ *kgo.Client, assigned map[string][]int32) {
			secondAssigned <- len(assigned[topic])
		}),
	)
	if err != nil {
		close(release)
		cancel()
		t.Fatal(err)
	}
	secondClosed := false
	defer func() {
		if !secondClosed {
			second.CloseAllowingRebalance()
		}
	}()
	secondPollCtx, stopSecondPoll := context.WithTimeout(ctx, 10*time.Second)
	secondPoll := make(chan *kgo.Record, 1)
	secondPollErr := make(chan error, 1)
	go func() {
		for {
			fetches := second.PollRecords(secondPollCtx, 1)
			if records := fetches.Records(); len(records) > 0 {
				secondPoll <- records[0]
				return
			}
			if errs := fetches.Errors(); len(errs) > 0 {
				secondPollErr <- fmt.Errorf("second consumer poll: %v", errs)
				return
			}
			if secondPollCtx.Err() != nil {
				secondPollErr <- secondPollCtx.Err()
				return
			}
		}
	}()

	select {
	case <-rebalanceBlocked:
	case <-ctx.Done():
		close(release)
		stopSecondPoll()
		cancel()
		t.Fatalf("second consumer did not trigger a blocked rebalance: %v", ctx.Err())
	}
	close(release)
	select {
	case err := <-runnerDone:
		if err != nil {
			stopSecondPoll()
			cancel()
			t.Fatalf("parallel batch: %v", err)
		}
	case <-ctx.Done():
		stopSecondPoll()
		cancel()
		t.Fatalf("parallel batch did not finish: %v", ctx.Err())
	}
	first.CloseAllowingRebalance()
	firstClosed = true

	assigned := false
	for !assigned {
		select {
		case count := <-secondAssigned:
			assigned = count > 0
		case <-ctx.Done():
			stopSecondPoll()
			cancel()
			t.Fatalf("second consumer did not receive a partition: %v", ctx.Err())
		}
	}
	select {
	case record := <-secondPoll:
		stopSecondPoll()
		cancel()
		t.Fatalf("second consumer redelivered committed record at partition %d offset %d", record.Partition, record.Offset)
	case err := <-secondPollErr:
		if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			stopSecondPoll()
			cancel()
			t.Fatalf("second consumer poll: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
	}
	stopSecondPoll()
	second.CloseAllowingRebalance()
	secondClosed = true
	cancel()
}

func TestKafkaTransactionAbortIsInvisibleAndCommitAdvancesInput(t *testing.T) {
	cfg := testConfig(t)
	input := newTopic(t, cfg, 1)
	output := newTopic(t, cfg, 1)
	group := newGroup(t, cfg)
	producer, err := kafka.NewProducer(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(producer.Close)
	first := orderEvent(uniqueID("evt"), 1250)
	second := orderEvent(uniqueID("evt"), 1750)
	produceEvent(t, cfg, producer, input, first)
	produceEvent(t, cfg, producer, input, second)

	transactionalID := uniqueID("kafka-lab-txn")
	session, err := kafka.NewTransactionalGroup(cfg, input, group, transactionalID)
	if err != nil {
		t.Fatal(err)
	}
	sessionClosed := false
	defer func() {
		if !sessionClosed {
			session.CloseAllowingRebalance()
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	err = kafka.ConsumeTransactionally(ctx, session, output, 1, true)
	cancel()
	if !errors.Is(err, kafka.ErrFailAfterTransactionalOutput) {
		t.Fatalf("aborted transaction error = %v, want failpoint", err)
	}

	visibleGroup := newGroup(t, cfg)
	visible, err := kafka.NewConsumer(cfg, output, visibleGroup, kgo.FetchIsolationLevel(kgo.ReadCommitted()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(visible.Close)
	ctx, cancel = context.WithTimeout(context.Background(), 500*time.Millisecond)
	record, pollErr := pollOne(ctx, visible)
	cancel()
	if record != nil || !errors.Is(pollErr, context.DeadlineExceeded) {
		t.Fatalf("read_committed after abort got record=%v err=%v, want no visible output", record, pollErr)
	}

	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	err = kafka.ConsumeTransactionally(ctx, session, output, 1, false)
	cancel()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	record, err = pollOne(ctx, visible)
	cancel()
	if err != nil {
		t.Fatalf("read committed first output: %v", err)
	}
	got, err := kafka.DecodeRecord(record)
	if err != nil || got.EventID != first.EventID {
		t.Fatalf("first committed output = %#v, err=%v; want event %q", got, err, first.EventID)
	}
	session.CloseAllowingRebalance()
	sessionClosed = true
	session, err = kafka.NewTransactionalGroup(cfg, input, group, transactionalID)
	if err != nil {
		t.Fatalf("recreate transactional group session: %v", err)
	}
	sessionClosed = false

	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	err = kafka.ConsumeTransactionally(ctx, session, output, 1, false)
	cancel()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	record, err = pollOne(ctx, visible)
	cancel()
	if err != nil {
		t.Fatalf("read committed second output: %v", err)
	}
	got, err = kafka.DecodeRecord(record)
	if err != nil || got.EventID != second.EventID {
		t.Fatalf("second committed output = %#v, err=%v; want event %q", got, err, second.EventID)
	}
}
