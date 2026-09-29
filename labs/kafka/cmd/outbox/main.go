package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/config"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/event"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/kafka"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/store"
	labsql "github.com/andreygilmiyarov/go-interview/labs/kafka/sql"
)

const (
	commandTimeout  = time.Minute
	cleanupTimeout  = 5 * time.Second
	maxPublishBatch = 1000
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		slog.Error("outbox command failed", "error", err)
		os.Exit(1)
	}
}

func run(parent context.Context, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: outbox <init|create|publish|consume> [flags]")
	}
	if args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		printUsage(os.Stdout)
		return nil
	}

	switch args[0] {
	case "init":
		flags := newFlags("init")
		timeout := addTimeoutFlag(flags)
		help, err := parseFlags(flags, args[1:])
		if err != nil {
			return err
		}
		if help {
			return nil
		}
		if flags.NArg() != 0 {
			return errors.New("init does not accept positional arguments")
		}
		ctx, cancel, err := commandContext(parent, *timeout)
		if err != nil {
			return err
		}
		defer cancel()
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		database, err := store.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer database.Close()
		if err := database.Init(ctx, labsql.Schema); err != nil {
			return err
		}
		slog.Info("outbox schema initialized", "command", "init", "database", "PostgreSQL")
		return nil
	case "create":
		flags := newFlags("create")
		timeout := addTimeoutFlag(flags)
		eventID := flags.String("event-id", "", "unique event ID")
		orderID := flags.String("order-id", "", "order ID")
		amountCents := flags.Int64("amount-cents", 0, "positive order amount in cents")
		help, err := parseFlags(flags, args[1:])
		if err != nil {
			return err
		}
		if help {
			return nil
		}
		if flags.NArg() != 0 {
			return errors.New("create does not accept positional arguments")
		}
		if strings.TrimSpace(*eventID) == "" || strings.TrimSpace(*orderID) == "" {
			return errors.New("create requires -event-id and -order-id")
		}
		orderEvent := event.Event{
			EventID:     *eventID,
			OrderID:     *orderID,
			Type:        "order.created",
			Version:     1,
			AmountCents: *amountCents,
		}
		if err := orderEvent.Validate(); err != nil {
			return fmt.Errorf("validate order: %w", err)
		}
		ctx, cancel, err := commandContext(parent, *timeout)
		if err != nil {
			return err
		}
		defer cancel()
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		database, err := store.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer database.Close()
		if err := database.CreateOrder(ctx, orderEvent); err != nil {
			return err
		}
		slog.Info("created order and outbox event", "command", "create", "event_id", orderEvent.EventID, "order_id", orderEvent.OrderID, "amount_cents", orderEvent.AmountCents)
		return nil
	case "publish":
		flags := newFlags("publish")
		timeout := addTimeoutFlag(flags)
		limit := flags.Int("limit", 10, "maximum pending events to publish (1-1000)")
		failAfterSend := flags.Bool("fail-after-send-before-mark", false, "simulate a crash after Kafka confirms a send and before PostgreSQL marks it")
		help, err := parseFlags(flags, args[1:])
		if err != nil {
			return err
		}
		if help {
			return nil
		}
		if flags.NArg() != 0 {
			return errors.New("publish does not accept positional arguments")
		}
		if *limit < 1 {
			return errors.New("publish limit must be positive")
		}
		if *limit > maxPublishBatch {
			return fmt.Errorf("publish limit must not exceed %d", maxPublishBatch)
		}
		ctx, cancel, err := commandContext(parent, *timeout)
		if err != nil {
			return err
		}
		defer cancel()
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		minISR := int16(2)
		if cfg.ReplicationFactor == 1 {
			minISR = 1
		}
		if err := kafka.CreateTopic(ctx, cfg, cfg.Topic, 3, minISR); err != nil {
			return err
		}
		database, err := store.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer database.Close()
		producer, err := kafka.NewProducer(cfg)
		if err != nil {
			return err
		}
		defer producer.Close()
		published, err := database.PublishPending(ctx, *limit, func(sendCtx context.Context, orderEvent event.Event) error {
			record, err := kafka.ProduceEvent(sendCtx, producer, cfg.Topic, orderEvent)
			if err == nil {
				slog.Info("Kafka confirmed outbox event", "command", "publish", "topic", record.Topic, "partition", record.Partition, "offset", record.Offset, "event_id", orderEvent.EventID, "order_id", orderEvent.OrderID)
			}
			return err
		}, *failAfterSend)
		if err != nil {
			return fmt.Errorf("publish outbox events: %w", err)
		}
		slog.Info("outbox publish batch completed", "command", "publish", "topic", cfg.Topic, "published", published)
		return nil
	case "consume":
		flags := newFlags("consume")
		timeout := addTimeoutFlag(flags)
		consumerID := flags.String("consumer", "", "Kafka group and PostgreSQL idempotency key")
		count := flags.Int("count", 1, "number of records to process before exiting")
		failBeforeCommit := flags.Bool("fail-after-process-before-commit", false, "simulate a crash after PostgreSQL commit and before Kafka offset commit")
		help, err := parseFlags(flags, args[1:])
		if err != nil {
			return err
		}
		if help {
			return nil
		}
		if flags.NArg() != 0 {
			return errors.New("consume does not accept positional arguments")
		}
		if strings.TrimSpace(*consumerID) == "" {
			return errors.New("consume requires -consumer")
		}
		if *count < 1 {
			return errors.New("consume count must be positive")
		}
		ctx, cancel, err := commandContext(parent, *timeout)
		if err != nil {
			return err
		}
		defer cancel()
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load configuration: %w", err)
		}
		database, err := store.Open(ctx, cfg.DatabaseURL)
		if err != nil {
			return err
		}
		defer database.Close()
		consumer, err := kafka.NewConsumer(
			cfg,
			cfg.Topic,
			*consumerID,
			kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
			kgo.FetchIsolationLevel(kgo.ReadCommitted()),
		)
		if err != nil {
			return err
		}
		defer func() {
			leaveCtx, leaveCancel := context.WithTimeout(context.Background(), cleanupTimeout)
			defer leaveCancel()
			if err := consumer.LeaveGroupContext(leaveCtx); err != nil {
				slog.Warn("leave Kafka consumer group", "group", *consumerID, "error", err)
			}
			consumer.Close()
		}()
		handler := func(ctx context.Context, _ *kgo.Record, orderEvent event.Event) error {
			applied, err := database.ApplyEvent(ctx, *consumerID, orderEvent)
			if err == nil {
				slog.Info("applied Kafka event to PostgreSQL", "command", "consume", "topic", cfg.Topic, "group", *consumerID, "event_id", orderEvent.EventID, "order_id", orderEvent.OrderID, "applied", applied)
			}
			return err
		}
		if err := kafka.ConsumeSequential(ctx, consumer, *count, *failBeforeCommit, handler); err != nil {
			return fmt.Errorf("consume order events: %w", err)
		}
		slog.Info("Kafka consume batch completed", "command", "consume", "topic", cfg.Topic, "group", *consumerID, "records", *count)
		return nil
	default:
		return fmt.Errorf("unknown outbox command %q (want init, create, publish, or consume)", args[0])
	}
}

func newFlags(name string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: outbox %s [flags]\n", name)
		flags.PrintDefaults()
	}
	return flags
}

func addTimeoutFlag(flags *flag.FlagSet) *time.Duration {
	return flags.Duration("timeout", commandTimeout, "maximum command duration (default 1m)")
}

func parseFlags(flags *flag.FlagSet, args []string) (bool, error) {
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func commandContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc, error) {
	if timeout <= 0 {
		return nil, nil, errors.New("command timeout must be positive")
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	return ctx, cancel, nil
}

func printUsage(writer *os.File) {
	fmt.Fprintln(writer, "Usage: outbox <init|create|publish|consume> [flags]")
	fmt.Fprintln(writer, "Subcommands: init, create, publish, consume")
	fmt.Fprintln(writer, "Use `outbox <subcommand> -h` to list subcommand flags.")
}
