package config

import "testing"

func TestLoadUsesDocumentedLocalDefaults(t *testing.T) {
	for _, key := range []string{"KAFKA_BROKERS", "KAFKA_TOPIC", "KAFKA_GROUP", "DATABASE_URL", "KAFKA_REPLICATION_FACTOR"} {
		t.Setenv(key, "")
	}

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Brokers) != 1 || got.Brokers[0] != "localhost:19092" {
		t.Fatalf("Brokers = %#v", got.Brokers)
	}
	if got.Topic != "orders" || got.Group != "orders-lab" {
		t.Fatalf("topic/group = %q/%q", got.Topic, got.Group)
	}
	if got.DatabaseURL != "postgres://kafka_lab:kafka_lab@localhost:15432/kafka_lab?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q", got.DatabaseURL)
	}
	if got.ReplicationFactor != 1 {
		t.Fatalf("ReplicationFactor = %d, want 1", got.ReplicationFactor)
	}
}

func TestLoadTrimsBrokerListAndReadsClusterReplicationFactor(t *testing.T) {
	t.Setenv("KAFKA_BROKERS", " broker-a:9092, broker-b:9092 ")
	t.Setenv("KAFKA_REPLICATION_FACTOR", "3")

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Brokers) != 2 || got.Brokers[0] != "broker-a:9092" || got.Brokers[1] != "broker-b:9092" {
		t.Fatalf("Brokers = %#v", got.Brokers)
	}
	if got.ReplicationFactor != 3 {
		t.Fatalf("ReplicationFactor = %d, want 3", got.ReplicationFactor)
	}
}

func TestLoadRejectsInvalidReplicationFactor(t *testing.T) {
	t.Setenv("KAFKA_REPLICATION_FACTOR", "0")
	if _, err := Load(); err == nil {
		t.Fatal("Load() accepted replication factor zero")
	}
}
