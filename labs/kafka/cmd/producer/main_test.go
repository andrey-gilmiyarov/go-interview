package main

import (
	"reflect"
	"testing"
	"time"

	"github.com/andreygilmiyarov/go-interview/labs/kafka/internal/config"
)

func TestParseFlagsUsesLabDefaults(t *testing.T) {
	got, err := parseFlags(nil, config.Config{Topic: "orders"})
	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}
	want := options{
		count:       1,
		topic:       "orders",
		amountCents: 12500,
		timeout:     60 * time.Second,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseFlags() = %#v, want %#v", got, want)
	}
}

func TestParseFlagsRejectsFixedEventIDForBatch(t *testing.T) {
	_, err := parseFlags([]string{"-count=2", "-event-id=evt-01"}, config.Config{Topic: "orders"})
	if err == nil {
		t.Fatal("parseFlags() accepted one event ID for multiple events")
	}
}

func TestBuildEventsAssignsUniqueGeneratedIDsAndOrderIDs(t *testing.T) {
	events, err := buildEvents(options{count: 3, topic: "orders", amountCents: 12500})
	if err != nil {
		t.Fatalf("buildEvents() error = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	eventIDs := map[string]bool{}
	orderIDs := map[string]bool{}
	for _, event := range events {
		if event.EventID == "" || eventIDs[event.EventID] {
			t.Fatalf("event ID %q is empty or duplicated", event.EventID)
		}
		if event.OrderID == "" || orderIDs[event.OrderID] {
			t.Fatalf("generated order ID %q is empty or duplicated", event.OrderID)
		}
		if event.Type != "order.created" || event.Version != 1 || event.AmountCents != 12500 {
			t.Fatalf("unexpected event fields: %#v", event)
		}
		eventIDs[event.EventID] = true
		orderIDs[event.OrderID] = true
	}
}

func TestBuildEventsKeepsExplicitOrderIDAcrossBatch(t *testing.T) {
	events, err := buildEvents(options{count: 2, topic: "orders", orderID: "order-01", amountCents: 12500})
	if err != nil {
		t.Fatalf("buildEvents() error = %v", err)
	}
	if len(events) != 2 || events[0].EventID == events[1].EventID {
		t.Fatalf("event IDs are not unique: %#v", events)
	}
	for _, event := range events {
		if event.OrderID != "order-01" {
			t.Fatalf("order ID = %q, want order-01", event.OrderID)
		}
	}
}

func TestBuildEventsPreservesSingleExplicitEventID(t *testing.T) {
	events, err := buildEvents(options{count: 1, topic: "orders", eventID: "evt-01", amountCents: 12500})
	if err != nil {
		t.Fatalf("buildEvents() error = %v", err)
	}
	if len(events) != 1 || events[0].EventID != "evt-01" {
		t.Fatalf("got events %#v, want one event with ID evt-01", events)
	}
}
