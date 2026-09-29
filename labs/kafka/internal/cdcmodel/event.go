package cdcmodel

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Event is the versioned JSON payload published by the CDC outbox.
type Event struct {
	EventID      string `json:"event_id"`
	OrderID      string `json:"order_id"`
	Type         string `json:"type"`
	Version      int    `json:"version"`
	OrderVersion int64  `json:"order_version"`
	AmountCents  int64  `json:"amount_cents"`
}

func (e Event) Validate() error {
	if strings.TrimSpace(e.EventID) == "" {
		return fmt.Errorf("event_id is required")
	}
	if strings.TrimSpace(e.OrderID) == "" {
		return fmt.Errorf("order_id is required")
	}
	if e.Version != 1 {
		return fmt.Errorf("version must be 1")
	}
	if e.OrderVersion < 1 {
		return fmt.Errorf("order_version must be positive")
	}
	switch e.Type {
	case "order.created":
		if e.OrderVersion != 1 {
			return fmt.Errorf("order.created must have order_version 1")
		}
	case "order.updated":
		if e.OrderVersion < 2 {
			return fmt.Errorf("order.updated must have order_version at least 2")
		}
	default:
		return fmt.Errorf("type must be order.created or order.updated")
	}
	if e.AmountCents <= 0 {
		return fmt.Errorf("amount_cents must be positive")
	}
	return nil
}

// Encode validates and emits the canonical field order for an event.
func Encode(event Event) ([]byte, error) {
	if err := event.Validate(); err != nil {
		return nil, fmt.Errorf("validate CDC event: %w", err)
	}
	body, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("encode CDC event: %w", err)
	}
	return body, nil
}

// Decode requires one valid JSON object with no unknown fields or trailing data.
func Decode(body []byte) (Event, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var event Event
	if err := decoder.Decode(&event); err != nil {
		return Event{}, fmt.Errorf("decode CDC event: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err == nil {
			return Event{}, fmt.Errorf("decode CDC event: multiple JSON values")
		}
		return Event{}, fmt.Errorf("decode CDC event trailing data: %w", err)
	}
	if err := event.Validate(); err != nil {
		return Event{}, fmt.Errorf("validate CDC event: %w", err)
	}
	return event, nil
}

// Fingerprint returns a stable SHA-256 hex digest over the typed JSON fields.
func Fingerprint(event Event) string {
	body, _ := json.Marshal(event)
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}
