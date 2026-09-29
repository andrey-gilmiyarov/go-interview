package sql

import _ "embed"

// Schema is the PostgreSQL schema used by the order and outbox lab.
//
//go:embed schema.sql
var Schema string
