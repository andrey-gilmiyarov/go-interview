package event

import "testing"

func TestEncodeProducesTheOrderEventContract(t *testing.T) {
	got, err := Encode(Event{
		EventID:     "evt-1",
		OrderID:     "order-1",
		Type:        "order.created",
		Version:     1,
		AmountCents: 1250,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"event_id":"evt-1","order_id":"order-1","type":"order.created","version":1,"amount_cents":1250}`
	if string(got) != want {
		t.Fatalf("Encode() = %s, want %s", got, want)
	}
}

func TestDecodeRejectsInvalidOrderEvents(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{"event_id":`},
		{name: "missing event ID", body: `{"order_id":"order-1","type":"order.created","version":1,"amount_cents":10}`},
		{name: "wrong event type", body: `{"event_id":"evt-1","order_id":"order-1","type":"order.updated","version":1,"amount_cents":10}`},
		{name: "unsupported version", body: `{"event_id":"evt-1","order_id":"order-1","type":"order.created","version":2,"amount_cents":10}`},
		{name: "non-positive amount", body: `{"event_id":"evt-1","order_id":"order-1","type":"order.created","version":1,"amount_cents":0}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Decode([]byte(tt.body)); err == nil {
				t.Fatal("Decode() accepted an invalid event")
			}
		})
	}
}

func TestDecodeAcceptsAndValidatesACompleteOrderEvent(t *testing.T) {
	got, err := Decode([]byte(`{"event_id":"evt-1","order_id":"order-1","type":"order.created","version":1,"amount_cents":1250}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.EventID != "evt-1" || got.OrderID != "order-1" || got.Type != "order.created" || got.Version != 1 || got.AmountCents != 1250 {
		t.Fatalf("Decode() = %#v", got)
	}
}
