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
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/kafka"
	"github.com/twmb/franz-go/pkg/kgo"
)

type options struct {
	inputTopic      string
	outputTopic     string
	group           string
	transactionalID string
	count           int
	failAfterOutput bool
	timeout         time.Duration
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
	if err := kafka.CreateTopic(ctx, cfg, opts.inputTopic, 3, minISR); err != nil {
		return err
	}
	if err := kafka.CreateTopic(ctx, cfg, opts.outputTopic, 3, minISR); err != nil {
		return err
	}
	session, err := kafka.NewTransactionalGroup(cfg, opts.inputTopic, opts.group, opts.transactionalID)
	if err != nil {
		return err
	}
	defer closeSession(session, logger)

	if err := kafka.ConsumeTransactionally(ctx, session, opts.outputTopic, opts.count, opts.failAfterOutput); err != nil {
		return err
	}
	logger.Info("transactional consumption complete",
		"input_topic", opts.inputTopic,
		"output_topic", opts.outputTopic,
		"group", opts.group,
		"transactional_id", opts.transactionalID,
		"count", opts.count,
	)
	return nil
}

func parseFlags(args []string, cfg config.Config) (options, error) {
	flags := flag.NewFlagSet("transactions", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var opts options
	flags.StringVar(&opts.inputTopic, "input-topic", cfg.Topic, "Kafka input topic (default KAFKA_TOPIC)")
	flags.StringVar(&opts.outputTopic, "output-topic", cfg.Topic+"-processed", "Kafka output topic")
	flags.StringVar(&opts.group, "group", cfg.Group, "consumer group (default KAFKA_GROUP)")
	flags.StringVar(&opts.transactionalID, "transactional-id", "", "stable default derives from group; set unique IDs for simultaneous instances")
	flags.IntVar(&opts.count, "count", 1, "number of input records to process")
	flags.BoolVar(&opts.failAfterOutput, "fail-after-output", false, "abort after enqueueing output and before transaction commit")
	flags.DurationVar(&opts.timeout, "timeout", 60*time.Second, "overall command timeout")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	opts.inputTopic = strings.TrimSpace(opts.inputTopic)
	opts.outputTopic = strings.TrimSpace(opts.outputTopic)
	opts.group = strings.TrimSpace(opts.group)
	opts.transactionalID = strings.TrimSpace(opts.transactionalID)
	if opts.inputTopic == "" || opts.outputTopic == "" || opts.group == "" {
		return options{}, fmt.Errorf("input-topic, output-topic, and group are required")
	}
	if opts.inputTopic == opts.outputTopic {
		return options{}, fmt.Errorf("input-topic and output-topic must be different")
	}
	if opts.transactionalID == "" {
		opts.transactionalID = opts.group + "-transactions"
	}
	if opts.count < 1 {
		return options{}, fmt.Errorf("count must be positive")
	}
	if opts.timeout <= 0 {
		return options{}, fmt.Errorf("timeout must be positive")
	}
	return opts, nil
}

func closeSession(session interface {
	Client() *kgo.Client
	CloseAllowingRebalance()
}, logger *slog.Logger) {
	leaveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := session.Client().LeaveGroupContext(leaveCtx); err != nil {
		logger.Warn("leaving transactional consumer group during shutdown", "error", err)
	}
	session.CloseAllowingRebalance()
}
