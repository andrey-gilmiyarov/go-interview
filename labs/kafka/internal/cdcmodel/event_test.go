package cdcmodel

import "testing"

func TestEncodeProducesCanonicalOrderEventJSON(t *testing.T) {
	event := Event{
		EventID:      "evt-1",
		OrderID:      "order-1",
		Type:         "order.created",
		Version:      1,
		OrderVersion: 1,
		AmountCents:  1250,
	}

	body, err := Encode(event)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"event_id":"evt-1","order_id":"order-1","type":"order.created","version":1,"order_version":1,"amount_cents":1250}`
	if string(body) != want {
		t.Fatalf("Encode() = %s, want %s", body, want)
	}
}

func TestDecodeRejectsMalformedAndNonStrictJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{"event_id":`},
		{name: "unknown field", body: `{"event_id":"evt-1","order_id":"order-1","type":"order.created","version":1,"order_version":1,"amount_cents":10,"extra":true}`},
		{name: "second JSON value", body: `{"event_id":"evt-1","order_id":"order-1","type":"order.created","version":1,"order_version":1,"amount_cents":10} {}`},
		{name: "trailing data", body: `{"event_id":"evt-1","order_id":"order-1","type":"order.created","version":1,"order_version":1,"amount_cents":10} garbage`},
		{name: "missing event ID", body: `{"order_id":"order-1","type":"order.created","version":1,"order_version":1,"amount_cents":10}`},
		{name: "wrong create version", body: `{"event_id":"evt-1","order_id":"order-1","type":"order.created","version":1,"order_version":2,"amount_cents":10}`},
		{name: "wrong update version", body: `{"event_id":"evt-1","order_id":"order-1","type":"order.updated","version":1,"order_version":1,"amount_cents":10}`},
		{name: "non-positive amount", body: `{"event_id":"evt-1","order_id":"order-1","type":"order.created","version":1,"order_version":1,"amount_cents":0}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Decode([]byte(test.body)); err == nil {
				t.Fatal("Decode() accepted invalid or non-strict JSON")
			}
		})
	}
}

func TestDecodePreservesIdentifierWhitespace(t *testing.T) {
	got, err := Decode([]byte(`{"event_id":" evt-1 ","order_id":" order-1 ","type":"order.created","version":1,"order_version":1,"amount_cents":1250}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.EventID != " evt-1 " || got.OrderID != " order-1 " {
		t.Fatalf("Decode() normalized identifiers: %#v", got)
	}
}

func TestFingerprintHashesCanonicalTypedEvent(t *testing.T) {
	got := Fingerprint(Event{
		EventID:      "evt-1",
		OrderID:      "order-1",
		Type:         "order.created",
		Version:      1,
		OrderVersion: 1,
		AmountCents:  1250,
	})
	want := "9846c76d0be32ce2982e676ebffea1e94f991fb8046c0d1ae67a3101ad57b159"
	if got != want {
		t.Fatalf("Fingerprint() = %q, want %q", got, want)
	}
}
