package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	defaultBrokers     = "localhost:19092"
	defaultTopic       = "orders"
	defaultGroup       = "orders-lab"
	defaultDatabaseURL = "postgres://kafka_lab:kafka_lab@localhost:15432/kafka_lab?sslmode=disable"
)

type Config struct {
	Brokers           []string
	Topic             string
	Group             string
	DatabaseURL       string
	ReplicationFactor int16
}

func Load() (Config, error) {
	brokers := splitBrokers(envOr("KAFKA_BROKERS", defaultBrokers))
	if len(brokers) == 0 {
		return Config{}, fmt.Errorf("KAFKA_BROKERS must contain at least one broker")
	}

	databaseURL := envOr("DATABASE_URL", defaultDatabaseURL)
	parsedURL, err := url.Parse(databaseURL)
	if err != nil || (parsedURL.Scheme != "postgres" && parsedURL.Scheme != "postgresql") || parsedURL.Host == "" {
		return Config{}, fmt.Errorf("DATABASE_URL must be a postgres URL with a host")
	}

	replicationFactor := int64(1)
	if value := strings.TrimSpace(os.Getenv("KAFKA_REPLICATION_FACTOR")); value != "" {
		replicationFactor, err = strconv.ParseInt(value, 10, 16)
		if err != nil || replicationFactor < 1 {
			return Config{}, fmt.Errorf("KAFKA_REPLICATION_FACTOR must be a positive integer")
		}
	}

	return Config{
		Brokers:           brokers,
		Topic:             envOr("KAFKA_TOPIC", defaultTopic),
		Group:             envOr("KAFKA_GROUP", defaultGroup),
		DatabaseURL:       databaseURL,
		ReplicationFactor: int16(replicationFactor),
	}, nil
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
