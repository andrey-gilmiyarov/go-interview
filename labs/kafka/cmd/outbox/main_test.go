package main

import (
	"context"
	"strings"
	"testing"
)

func TestPublishRejectsBatchAboveCapBeforeConnecting(t *testing.T) {
	err := run(context.Background(), []string{"publish", "-limit=1001"})
	if err == nil || !strings.Contains(err.Error(), "must not exceed 1000") {
		t.Fatalf("run() error = %v, want publish limit cap diagnostic", err)
	}
}

func TestSubcommandHelpIsSuccessful(t *testing.T) {
	if err := run(context.Background(), []string{"publish", "-h"}); err != nil {
		t.Fatalf("run() error = %v, want help to exit successfully", err)
	}
}

func TestCommandTimeoutMustBePositive(t *testing.T) {
	err := run(context.Background(), []string{"init", "-timeout=0s"})
	if err == nil || !strings.Contains(err.Error(), "timeout must be positive") {
		t.Fatalf("run() error = %v, want positive timeout diagnostic", err)
	}
}
