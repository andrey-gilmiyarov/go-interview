package kafka

import (
	"context"
	"errors"
	"testing"

	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/event"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestCreateTopicsRequestAppliesReplicationAndMinimumISR(t *testing.T) {
	request, err := createTopicsRequest("orders-integration", 3, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(request.Topics) != 1 {
		t.Fatalf("topic count = %d, want 1", len(request.Topics))
	}
	topic := request.Topics[0]
	if topic.Topic != "orders-integration" || topic.NumPartitions != 3 || topic.ReplicationFactor != 3 {
		t.Fatalf("topic request = %#v", topic)
	}
	var minISR *string
	for i := range topic.Configs {
		if topic.Configs[i].Name == "min.insync.replicas" {
			minISR = topic.Configs[i].Value
		}
	}
	if minISR == nil || *minISR != "2" {
		t.Fatalf("min.insync.replicas = %v, want 2", minISR)
	}
}

func TestCreateTopicsRequestAllowsNoExplicitMinimumISR(t *testing.T) {
	request, err := createTopicsRequest("orders-local", 1, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := request.Topics[0].ReplicationFactor; got != 1 {
		t.Fatalf("replication factor = %d, want 1", got)
	}
	for _, cfg := range request.Topics[0].Configs {
		if cfg.Name == "min.insync.replicas" {
			t.Fatal("request unexpectedly set min.insync.replicas")
		}
	}
}

func TestCreateTopicsRequestRejectsInvalidReplicationShape(t *testing.T) {
	tests := []struct {
		name              string
		topic             string
		partitions        int32
		replicationFactor int16
		minISR            int16
	}{
		{name: "empty topic", topic: "", partitions: 1, replicationFactor: 1},
		{name: "no partitions", topic: "orders", partitions: 0, replicationFactor: 1},
		{name: "no replicas", topic: "orders", partitions: 1, replicationFactor: 0},
		{name: "minimum above replicas", topic: "orders", partitions: 1, replicationFactor: 2, minISR: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := createTopicsRequest(tt.topic, tt.partitions, tt.replicationFactor, tt.minISR); err == nil {
				t.Fatal("createTopicsRequest() accepted an invalid topic shape")
			}
		})
	}
}

func TestDeleteTopicsRequestNamesTopicForLegacyAndCurrentVersions(t *testing.T) {
	request, err := deleteTopicsRequest("owned-test-topic")
	if err != nil {
		t.Fatal(err)
	}
	if len(request.TopicNames) != 1 || request.TopicNames[0] != "owned-test-topic" {
		t.Fatalf("legacy topic names = %#v", request.TopicNames)
	}
	if len(request.Topics) != 1 || request.Topics[0].Topic == nil || *request.Topics[0].Topic != "owned-test-topic" {
		t.Fatalf("current topic request = %#v", request.Topics)
	}
}

func TestDeleteTopicsRequestRejectsEmptyTopic(t *testing.T) {
	if _, err := deleteTopicsRequest(""); err == nil {
		t.Fatal("deleteTopicsRequest() accepted an empty topic name")
	}
}

func TestDeleteGroupsRequestNamesExactlyRequestedGroup(t *testing.T) {
	request, err := deleteGroupsRequest("orders-test-group")
	if err != nil {
		t.Fatal(err)
	}
	if len(request.Groups) != 1 || request.Groups[0] != "orders-test-group" {
		t.Fatalf("groups = %#v", request.Groups)
	}
}

func TestDeleteGroupsRequestRejectsEmptyGroup(t *testing.T) {
	if _, err := deleteGroupsRequest(""); err == nil {
		t.Fatal("deleteGroupsRequest() accepted an empty group")
	}
}

func TestDecodeRecordRejectsKeyThatDoesNotMatchOrderID(t *testing.T) {
	record := &kgo.Record{
		Key:   []byte("different-order"),
		Value: []byte("{\"event_id\":\"evt-1\",\"order_id\":\"order-1\",\"type\":\"order.created\",\"version\":1,\"amount_cents\":1250}"),
	}
	if _, err := DecodeRecord(record); err == nil {
		t.Fatal("DecodeRecord() accepted a key that can route this event to the wrong order partition")
	}
}

func TestProduceEventValidatesBeforeUsingKafkaClient(t *testing.T) {
	invalid := event.Event{EventID: "evt-1", OrderID: "order-1", Type: "order.created", Version: 1}
	if _, err := ProduceEvent(context.Background(), nil, "orders", invalid); err == nil {
		t.Fatal("ProduceEvent() accepted an invalid order amount")
	}
}

func TestDeliveryCallbackAlreadyReadyWinsOverCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	want := &kgo.Record{Topic: "orders", Partition: 2, Offset: 17}
	delivered := make(chan deliveryResult, 1)
	delivered <- deliveryResult{record: want}

	got, err := awaitDelivery(ctx, delivered, "evt-1", "orders")
	if err != nil {
		t.Fatalf("awaitDelivery() error = %v, want confirmed delivery", err)
	}
	if got != want {
		t.Fatalf("awaitDelivery() record = %p, want %p", got, want)
	}
}

func TestDeliveryWithoutReadyCallbackKeepsUnknownOutcome(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	delivered := make(chan deliveryResult, 1)

	if _, err := awaitDelivery(ctx, delivered, "evt-1", "orders"); !errors.Is(err, ErrDeliveryOutcomeUnknown) {
		t.Fatalf("awaitDelivery() error = %v, want ErrDeliveryOutcomeUnknown", err)
	}
}
