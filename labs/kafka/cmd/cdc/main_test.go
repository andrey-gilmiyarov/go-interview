package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestParseCreateAndUpdateCommands(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		wantType        string
		wantVersion     int64
		wantAmountCents int64
	}{
		{
			name:            "create",
			args:            []string{"create", "-event-id", "evt-1", "-order-id", "order-1", "-amount-cents", "1000"},
			wantType:        "order.created",
			wantAmountCents: 1000,
		},
		{
			name:            "update",
			args:            []string{"update", "-event-id", "evt-2", "-order-id", "order-1", "-amount-cents", "1500", "-expected-version", "1"},
			wantType:        "order.updated",
			wantVersion:     1,
			wantAmountCents: 1500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInvocation(tt.args)
			if err != nil {
				t.Fatal(err)
			}
			if got.command.Type != tt.wantType || got.command.ExpectedVersion != tt.wantVersion || got.command.AmountCents != tt.wantAmountCents {
				t.Fatalf("command = %#v; want type=%q expected_version=%d amount_cents=%d", got.command, tt.wantType, tt.wantVersion, tt.wantAmountCents)
			}
		})
	}
}

func TestParseRejectsInvalidOrUnknownCommandArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing event ID", args: []string{"create", "-order-id", "order-1", "-amount-cents", "1000"}},
		{name: "nonpositive amount", args: []string{"create", "-event-id", "evt-1", "-order-id", "order-1", "-amount-cents", "0"}},
		{name: "create rejects update version flag", args: []string{"create", "-event-id", "evt-1", "-order-id", "order-1", "-amount-cents", "1000", "-expected-version", "1"}},
		{name: "update requires expected version", args: []string{"update", "-event-id", "evt-1", "-order-id", "order-1", "-amount-cents", "1000"}},
		{name: "unknown option", args: []string{"create", "-event-id", "evt-1", "-order-id", "order-1", "-amount-cents", "1000", "-mystery"}},
		{name: "unexpected positional argument", args: []string{"order", "-order-id", "order-1", "extra"}},
		{name: "consume requires projection", args: []string{"consume"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseInvocation(tt.args); err == nil {
				t.Fatal("parseInvocation() accepted invalid arguments")
			}
		})
	}
}

func TestRunHelpPrintsUsageAndReturnsSuccess(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if err := run(context.Background(), []string{"--help"}, &stdout, &stderr); err != nil {
		t.Fatalf("run help: %v", err)
	}
	if !strings.Contains(stdout.String(), "cdc <create|update|order|consume|projection>") || stderr.Len() != 0 {
		t.Fatalf("help stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunSubcommandHelpListsItsFlags(t *testing.T) {
	tests := []struct {
		action string
		flag   string
	}{
		{action: "create", flag: "-event-id"},
		{action: "update", flag: "-expected-version"},
		{action: "order", flag: "-order-id"},
		{action: "consume", flag: "-fail-after-process-before-commit"},
		{action: "projection", flag: "-projection"},
	}
	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if err := run(context.Background(), []string{tt.action, "-h"}, &stdout, &stderr); err != nil {
				t.Fatalf("run %s help: %v", tt.action, err)
			}
			if !strings.Contains(stdout.String(), tt.flag) || stderr.Len() != 0 {
				t.Fatalf("%s help stdout=%q stderr=%q", tt.action, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunValidatesBeforeOpeningDatabase(t *testing.T) {
	t.Setenv("CDC_DATABASE_URL", "postgres://cdc_app:cdc_app@127.0.0.1:1/cdc_lab?sslmode=disable")
	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{"create", "-event-id", "evt-1", "-order-id", "order-1", "-amount-cents", "0"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "amount-cents") {
		t.Fatalf("run invalid command error = %v, want amount-cents validation before database access", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("invalid command wrote result to stdout: %q", stdout.String())
	}
}

func TestRunRejectsMaximumOrderVersionBeforeOpeningDatabase(t *testing.T) {
	t.Setenv("CDC_DATABASE_URL", "postgres://cdc_app:cdc_app@127.0.0.1:1/cdc_lab?sslmode=disable")
	var stdout, stderr bytes.Buffer
	err := run(context.Background(), []string{"update", "-event-id", "evt-1", "-order-id", "order-1", "-amount-cents", "1000", "-expected-version", "9223372036854775807"}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "expected-version") {
		t.Fatalf("run maximum version error = %v, want validation before database access", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("invalid maximum version wrote result to stdout: %q", stdout.String())
	}
}
