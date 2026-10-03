package lab

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestRetryOnlyKnownAbort(t *testing.T) {
	for _, tc := range []struct {
		name  string
		err   error
		calls int
	}{
		{"serialization", &pgconn.PgError{Code: "40001"}, 3},
		{"deadlock", &pgconn.PgError{Code: "40P01"}, 3},
		{"unknown commit", io.ErrUnexpectedEOF, 1},
		{"constraint", &pgconn.PgError{Code: "23505"}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := 0
			err := Retry(context.Background(), 3, func(context.Context) error { n++; return tc.err })
			if !errors.Is(err, tc.err) || n != tc.calls {
				t.Fatalf("calls=%d err=%v", n, err)
			}
		})
	}
}
func TestRetryCancelledAndSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	n := 0
	if err := Retry(ctx, 3, func(context.Context) error { n++; return nil }); !errors.Is(err, context.Canceled) || n != 0 {
		t.Fatalf("calls=%d err=%v", n, err)
	}
	n = 0
	if err := Retry(context.Background(), 3, func(context.Context) error {
		n++
		if n == 1 {
			return &pgconn.PgError{Code: "40001"}
		}
		return nil
	}); err != nil || n != 2 {
		t.Fatalf("calls=%d err=%v", n, err)
	}
}
