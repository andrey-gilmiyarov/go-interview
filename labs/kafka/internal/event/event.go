package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Event struct {
	EventID     string `json:"event_id"`
	OrderID     string `json:"order_id"`
	Type        string `json:"type"`
	Version     int    `json:"version"`
	AmountCents int64  `json:"amount_cents"`
}

func (e Event) Validate() error {
	if strings.TrimSpace(e.EventID) == "" {
		return fmt.Errorf("event_id is required")
	}
	if strings.TrimSpace(e.OrderID) == "" {
		return fmt.Errorf("order_id is required")
	}
	if e.Type != "order.created" {
		return fmt.Errorf("type must be order.created")
	}
	if e.Version != 1 {
		return fmt.Errorf("version must be 1")
	}
	if e.AmountCents <= 0 {
		return fmt.Errorf("amount_cents must be positive")
	}
	return nil
}

func Encode(e Event) ([]byte, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}
	body, err := json.Marshal(e)
	if err != nil {
		return nil, fmt.Errorf("encode order event: %w", err)
	}
	return body, nil
}

func Decode(body []byte) (Event, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var e Event
	if err := decoder.Decode(&e); err != nil {
		return Event{}, fmt.Errorf("decode order event: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err == nil {
			return Event{}, fmt.Errorf("decode order event: multiple JSON values")
		}
		return Event{}, fmt.Errorf("decode order event trailing data: %w", err)
	}
	if err := e.Validate(); err != nil {
		return Event{}, fmt.Errorf("validate order event: %w", err)
	}
	return e, nil
}
