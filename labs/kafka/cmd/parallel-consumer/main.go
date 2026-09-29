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

	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/config"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/event"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/kafka"
	"github.com/twmb/franz-go/pkg/kgo"
)

type options struct {
	count                        int
	topic                        string
	group                        string
	failAfterProcessBeforeCommit bool
	timeout                      time.Duration
}

func main() {
	if err := execute(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func execute(args []string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	return run(ctx, args, logger)
}

func run(parent context.Context, args []string, logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	opts, err := parseFlags(args, cfg)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, opts.timeout)
	defer cancel()
	minISR := int16(2)
	if cfg.ReplicationFactor == 1 {
		minISR = 1
	}
	if err := kafka.CreateTopic(ctx, cfg, opts.topic, 3, minISR); err != nil {
		return err
	}
	client, err := kafka.NewConsumer(cfg, opts.topic, opts.group, kgo.FetchIsolationLevel(kgo.ReadCommitted()), kgo.BlockRebalanceOnPoll())
	if err != nil {
		return err
	}
	defer closeConsumer(client, logger)

	handler := func(handlerCtx context.Context, record *kgo.Record, received event.Event) error {
		if err := handlerCtx.Err(); err != nil {
			return err
		}
		logger.Info("consumed order event in parallel batch",
			"topic", record.Topic,
			"partition", record.Partition,
			"offset", record.Offset,
			"event_id", received.EventID,
			"order_id", received.OrderID,
		)
		return nil
	}
	if err := kafka.ConsumeParallel(ctx, client, opts.count, opts.failAfterProcessBeforeCommit, handler); err != nil {
		return err
	}
	logger.Info("parallel consumption complete", "topic", opts.topic, "group", opts.group, "count", opts.count)
	return nil
}

func parseFlags(args []string, cfg config.Config) (options, error) {
	flags := flag.NewFlagSet("parallel-consumer", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var opts options
	flags.IntVar(&opts.count, "count", 1, "number of records to consume")
	flags.StringVar(&opts.topic, "topic", cfg.Topic, "Kafka topic (default KAFKA_TOPIC)")
	flags.StringVar(&opts.group, "group", cfg.Group, "consumer group (default KAFKA_GROUP)")
	flags.BoolVar(&opts.failAfterProcessBeforeCommit, "fail-after-process-before-commit", false, "inject one failure after processing and before offset commit")
	flags.DurationVar(&opts.timeout, "timeout", 60*time.Second, "overall command timeout")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	opts.topic = strings.TrimSpace(opts.topic)
	opts.group = strings.TrimSpace(opts.group)
	if opts.count < 1 {
		return options{}, fmt.Errorf("count must be positive")
	}
	if opts.topic == "" || opts.group == "" {
		return options{}, fmt.Errorf("topic and group are required")
	}
	if opts.timeout <= 0 {
		return options{}, fmt.Errorf("timeout must be positive")
	}
	return opts, nil
}

func closeConsumer(client *kgo.Client, logger *slog.Logger) {
	leaveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.LeaveGroupContext(leaveCtx); err != nil {
		logger.Warn("leaving consumer group during shutdown", "error", err)
	}
	client.CloseAllowingRebalance()
}
