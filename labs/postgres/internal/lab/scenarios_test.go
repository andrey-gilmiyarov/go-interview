//go:build integration

package lab

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestScenarios(t *testing.T) {
	for _, id := range IDs {
		t.Run(id, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			var out bytes.Buffer
			if err := Run(ctx, id, &out); err != nil {
				t.Fatalf("%v\n%s", err, out.String())
			}
			if !strings.Contains(out.String(), "PASS "+id) {
				t.Fatal(out.String())
			}
			if (id == "stock-race" || id == "idempotency") && !strings.Contains(out.String(), "Broken:") {
				t.Fatal("missing reproducible broken variant")
			}
			t.Log(out.String())
		})
	}
}
func TestUnknownScenario(t *testing.T) {
	if err := Run(context.Background(), "missing", &bytes.Buffer{}); err == nil {
		t.Fatal("unknown scenario accepted")
	}
}
