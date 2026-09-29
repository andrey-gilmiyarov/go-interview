package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/cdcconsumer"
	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/cdcmodel"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	defaultDatabaseURL = "postgres://cdc_app:cdc_app@127.0.0.1:25432/cdc_lab?sslmode=disable"
	defaultBrokers     = "127.0.0.1:49092"
	defaultTopic       = "cdc.orders"
	defaultTimeout     = time.Minute
	cleanupTimeout     = 5 * time.Second
)

type invocation struct {
	action           string
	databaseURL      string
	brokers          []string
	topic            string
	command          cdcmodel.Command
	orderID          string
	projection       string
	count            int
	failBeforeCommit bool
	timeout          time.Duration
	help             bool
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		slog.Error("CDC command failed", "error", err)
		os.Exit(1)
	}
}

func run(parent context.Context, args []string, stdout, stderr io.Writer) error {
	parsed, err := parseInvocation(args)
	if err != nil {
		if len(args) == 0 {
			printUsage(stderr, "")
		}
		return err
	}
	if parsed.help {
		printUsage(stdout, parsed.action)
		return nil
	}
	if err := validateDatabaseURL(parsed.databaseURL); err != nil {
		return err
	}
	if parsed.timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	ctx, cancel := context.WithTimeout(parent, parsed.timeout)
	defer cancel()

	if parsed.action == "consume" {
		return runConsume(ctx, parsed, stdout)
	}
	store, err := cdcmodel.Open(ctx, parsed.databaseURL)
	if err != nil {
		return fmt.Errorf("open CDC model database: %w", err)
	}
	defer store.Close()

	var result any
	switch parsed.action {
	case "create", "update":
		result, err = store.Execute(ctx, parsed.command)
	case "order":
		result, err = store.GetOrder(ctx, parsed.orderID)
	case "projection":
		result, err = store.GetProjection(ctx, parsed.projection, parsed.orderID)
	default:
		return fmt.Errorf("unknown command %q", parsed.action)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", parsed.action, err)
	}
	return writeJSON(stdout, result)
}

func runConsume(ctx context.Context, parsed invocation, stdout io.Writer) error {
	if len(parsed.brokers) == 0 {
		return fmt.Errorf("CDC_KAFKA_BROKERS must contain at least one broker")
	}
	if strings.TrimSpace(parsed.topic) == "" {
		return fmt.Errorf("CDC_TOPIC is required")
	}
	store, err := cdcmodel.Open(ctx, parsed.databaseURL)
	if err != nil {
		return fmt.Errorf("open CDC model database: %w", err)
	}
	defer store.Close()

	group := "cdc." + parsed.projection
	client, err := kgo.NewClient(
		kgo.SeedBrokers(parsed.brokers...),
		kgo.ClientID("go-interview-cdc-consumer"),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(parsed.topic),
		kgo.ConsumeStartOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
		kgo.FetchIsolationLevel(kgo.ReadCommitted()),
		kgo.FetchMaxWait(200*time.Millisecond),
		kgo.BlockRebalanceOnPoll(),
	)
	if err != nil {
		return fmt.Errorf("create CDC consumer for topic %q group %q: %w", parsed.topic, group, err)
	}
	defer func() {
		leaveCtx, leaveCancel := context.WithTimeout(context.Background(), cleanupTimeout)
		defer leaveCancel()
		if err := client.LeaveGroupContext(leaveCtx); err != nil {
			slog.Warn("leave CDC consumer group", "group", group, "error", err)
		}
		client.CloseAllowingRebalance()
	}()
	if err := cdcconsumer.Consume(ctx, client, store, parsed.projection, parsed.count, parsed.failBeforeCommit); err != nil {
		return fmt.Errorf("consume CDC events: %w", err)
	}
	return writeJSON(stdout, map[string]any{
		"command":    "consume",
		"projection": parsed.projection,
		"group":      group,
		"topic":      parsed.topic,
		"records":    parsed.count,
	})
}

func parseInvocation(args []string) (invocation, error) {
	if len(args) == 0 {
		return invocation{}, errors.New("usage: cdc <create|update|order|consume|projection> [flags]")
	}
	action := args[0]
	if action == "help" || action == "-h" || action == "--help" {
		return invocation{action: "", help: true}, nil
	}
	parsed := invocation{
		action:      action,
		databaseURL: envOr("CDC_DATABASE_URL", defaultDatabaseURL),
		brokers:     splitBrokers(envOr("CDC_KAFKA_BROKERS", defaultBrokers)),
		topic:       envOr("CDC_TOPIC", defaultTopic),
		count:       1,
		timeout:     defaultTimeout,
	}
	flags := flag.NewFlagSet("cdc "+action, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Usage = func() { printUsage(io.Discard, action) }
	timeout := flags.Duration("timeout", defaultTimeout, "maximum command duration (default 1m)")

	switch action {
	case "create", "update":
		eventID := flags.String("event-id", "", "idempotency key for this command")
		orderID := flags.String("order-id", "", "order ID")
		amount := flags.Int64("amount-cents", 0, "positive absolute amount in cents")
		var expectedVersion int64
		if action == "update" {
			flags.Int64Var(&expectedVersion, "expected-version", 0, "expected current order version")
		}
		if err := parseFlags(flags, args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				parsed.help = true
				return parsed, nil
			}
			return invocation{}, err
		}
		if flags.NArg() != 0 {
			return invocation{}, fmt.Errorf("%s does not accept positional arguments", action)
		}
		if strings.TrimSpace(*eventID) == "" || strings.TrimSpace(*orderID) == "" {
			return invocation{}, fmt.Errorf("%s requires -event-id and -order-id", action)
		}
		if *amount <= 0 {
			return invocation{}, fmt.Errorf("amount-cents must be positive")
		}
		typeName := "order.created"
		if action == "update" {
			typeName = "order.updated"
			if expectedVersion <= 0 {
				return invocation{}, fmt.Errorf("update requires a positive -expected-version")
			}
			if expectedVersion == math.MaxInt64 {
				return invocation{}, fmt.Errorf("expected-version is too large to increment")
			}
		}
		parsed.command = cdcmodel.Command{
			EventID:         *eventID,
			OrderID:         *orderID,
			Type:            typeName,
			ExpectedVersion: expectedVersion,
			AmountCents:     *amount,
		}
	case "order":
		orderID := flags.String("order-id", "", "order ID")
		if err := parseFlags(flags, args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				parsed.help = true
				return parsed, nil
			}
			return invocation{}, err
		}
		if flags.NArg() != 0 {
			return invocation{}, errors.New("order does not accept positional arguments")
		}
		if strings.TrimSpace(*orderID) == "" {
			return invocation{}, errors.New("order requires -order-id")
		}
		parsed.orderID = *orderID
	case "consume":
		projection := flags.String("projection", "", "projection name; Kafka group is cdc.<projection>")
		count := flags.Int("count", 1, "number of Kafka records to process")
		fail := flags.Bool("fail-after-process-before-commit", false, "fail after PostgreSQL commit and before Kafka offset commit")
		if err := parseFlags(flags, args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				parsed.help = true
				return parsed, nil
			}
			return invocation{}, err
		}
		if flags.NArg() != 0 {
			return invocation{}, errors.New("consume does not accept positional arguments")
		}
		if strings.TrimSpace(*projection) == "" {
			return invocation{}, errors.New("consume requires -projection")
		}
		if *count < 1 {
			return invocation{}, errors.New("consume count must be positive")
		}
		parsed.projection, parsed.count, parsed.failBeforeCommit = *projection, *count, *fail
	case "projection":
		projection := flags.String("projection", "", "projection name")
		orderID := flags.String("order-id", "", "order ID")
		if err := parseFlags(flags, args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				parsed.help = true
				return parsed, nil
			}
			return invocation{}, err
		}
		if flags.NArg() != 0 {
			return invocation{}, errors.New("projection does not accept positional arguments")
		}
		if strings.TrimSpace(*projection) == "" || strings.TrimSpace(*orderID) == "" {
			return invocation{}, errors.New("projection requires -projection and -order-id")
		}
		parsed.projection, parsed.orderID = *projection, *orderID
	default:
		return invocation{}, fmt.Errorf("unknown command %q", action)
	}
	parsed.timeout = *timeout
	if parsed.timeout <= 0 {
		return invocation{}, errors.New("timeout must be positive")
	}
	return parsed, nil
}

func parseFlags(flags *flag.FlagSet, args []string) error {
	if err := flags.Parse(args); err != nil {
		return err
	}
	return nil
}

func validateDatabaseURL(databaseURL string) error {
	parsed, err := url.Parse(databaseURL)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" {
		return errors.New("CDC_DATABASE_URL must be a PostgreSQL URL with a host")
	}
	return nil
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

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func writeJSON(writer io.Writer, value any) error {
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		return fmt.Errorf("write JSON result: %w", err)
	}
	return nil
}

func printUsage(writer io.Writer, action string) {
	if action == "" {
		fmt.Fprintln(writer, "Usage: cdc <create|update|order|consume|projection> [flags]")
		fmt.Fprintln(writer, "Use `cdc <subcommand> -h` to list subcommand flags.")
		return
	}
	fmt.Fprintf(writer, "Usage: cdc %s [flags]\n", action)
	switch action {
	case "create":
		fmt.Fprintln(writer, "  -event-id string       idempotency key for this command")
		fmt.Fprintln(writer, "  -order-id string       order ID")
		fmt.Fprintln(writer, "  -amount-cents int64    positive absolute amount in cents")
		fmt.Fprintln(writer, "  -timeout duration     maximum command duration (default 1m)")
	case "update":
		fmt.Fprintln(writer, "  -event-id string       idempotency key for this command")
		fmt.Fprintln(writer, "  -order-id string       order ID")
		fmt.Fprintln(writer, "  -amount-cents int64    positive absolute amount in cents")
		fmt.Fprintln(writer, "  -expected-version int64 expected current order version")
		fmt.Fprintln(writer, "  -timeout duration     maximum command duration (default 1m)")
	case "order":
		fmt.Fprintln(writer, "  -order-id string   order ID")
		fmt.Fprintln(writer, "  -timeout duration maximum command duration (default 1m)")
	case "consume":
		fmt.Fprintln(writer, "  -projection string                         projection name; Kafka group is cdc.<projection>")
		fmt.Fprintln(writer, "  -count int                                 number of Kafka records to process (default 1)")
		fmt.Fprintln(writer, "  -fail-after-process-before-commit          fail after PostgreSQL commit and before Kafka offset commit")
		fmt.Fprintln(writer, "  -timeout duration                         maximum command duration (default 1m)")
	case "projection":
		fmt.Fprintln(writer, "  -projection string projection name")
		fmt.Fprintln(writer, "  -order-id string  order ID")
		fmt.Fprintln(writer, "  -timeout duration maximum command duration (default 1m)")
	}
	fmt.Fprintln(writer, "Environment: CDC_DATABASE_URL, CDC_KAFKA_BROKERS, CDC_TOPIC")
}
