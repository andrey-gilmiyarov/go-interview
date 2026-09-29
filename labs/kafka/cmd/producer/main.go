package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
)

type options struct {
	count       int
	topic       string
	eventID     string
	orderID     string
	amountCents int64
	timeout     time.Duration
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
	events, err := buildEvents(opts)
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
	client, err := kafka.NewProducer(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	for _, orderEvent := range events {
		record, err := kafka.ProduceEvent(ctx, client, opts.topic, orderEvent)
		if err != nil {
			return err
		}
		logger.Info("produced order event",
			"topic", record.Topic,
			"partition", record.Partition,
			"offset", record.Offset,
			"event_id", orderEvent.EventID,
			"order_id", orderEvent.OrderID,
		)
	}
	return nil
}

func parseFlags(args []string, cfg config.Config) (options, error) {
	flags := flag.NewFlagSet("producer", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var opts options
	flags.IntVar(&opts.count, "count", 1, "number of order events to produce")
	flags.StringVar(&opts.topic, "topic", cfg.Topic, "Kafka topic (default KAFKA_TOPIC)")
	flags.StringVar(&opts.eventID, "event-id", "", "event ID for one event; generated uniquely when omitted")
	flags.StringVar(&opts.orderID, "order-id", "", "order ID shared by the batch; generated per event when omitted")
	flags.Int64Var(&opts.amountCents, "amount-cents", 12500, "positive order amount in cents")
	flags.DurationVar(&opts.timeout, "timeout", 60*time.Second, "overall command timeout")
	if err := flags.Parse(args); err != nil {
		return options{}, err
	}
	if flags.NArg() != 0 {
		return options{}, fmt.Errorf("unexpected positional arguments: %s", strings.Join(flags.Args(), " "))
	}
	opts.topic = strings.TrimSpace(opts.topic)
	opts.eventID = strings.TrimSpace(opts.eventID)
	opts.orderID = strings.TrimSpace(opts.orderID)
	if opts.count < 1 {
		return options{}, fmt.Errorf("count must be positive")
	}
	if opts.topic == "" {
		return options{}, fmt.Errorf("topic is required")
	}
	if opts.amountCents <= 0 {
		return options{}, fmt.Errorf("amount-cents must be positive")
	}
	if opts.eventID != "" && opts.count > 1 {
		return options{}, fmt.Errorf("event-id can be supplied only when count is 1; batch event IDs are generated uniquely")
	}
	if opts.timeout <= 0 {
		return options{}, fmt.Errorf("timeout must be positive")
	}
	return opts, nil
}

func buildEvents(opts options) ([]event.Event, error) {
	if opts.count < 1 {
		return nil, fmt.Errorf("count must be positive")
	}
	if opts.amountCents <= 0 {
		return nil, fmt.Errorf("amount-cents must be positive")
	}
	if opts.eventID != "" && opts.count > 1 {
		return nil, fmt.Errorf("event-id can be supplied only when count is 1")
	}

	result := make([]event.Event, 0, opts.count)
	seenEventIDs := make(map[string]struct{}, opts.count)
	for range opts.count {
		eventID := opts.eventID
		suffix := ""
		if eventID == "" {
			var err error
			eventID, suffix, err = uniqueEventID(seenEventIDs)
			if err != nil {
				return nil, err
			}
		}
		orderID := opts.orderID
		if orderID == "" {
			if suffix == "" {
				var err error
				suffix, err = randomSuffix()
				if err != nil {
					return nil, err
				}
			}
			orderID = "order-" + suffix
		}
		orderEvent := event.Event{
			EventID:     eventID,
			OrderID:     orderID,
			Type:        "order.created",
			Version:     1,
			AmountCents: opts.amountCents,
		}
		if err := orderEvent.Validate(); err != nil {
			return nil, err
		}
		result = append(result, orderEvent)
	}
	return result, nil
}

func uniqueEventID(seen map[string]struct{}) (string, string, error) {
	for {
		suffix, err := randomSuffix()
		if err != nil {
			return "", "", err
		}
		eventID := "evt-" + suffix
		if _, exists := seen[eventID]; exists {
			continue
		}
		seen[eventID] = struct{}{}
		return eventID, suffix, nil
	}
}

func randomSuffix() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate event identifier: %w", err)
	}
	return hex.EncodeToString(value[:]), nil
}
