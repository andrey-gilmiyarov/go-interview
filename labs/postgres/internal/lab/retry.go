package lab

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// Retry reruns the WHOLE transaction callback, never an individual statement.
// Transport errors (including an unknown COMMIT outcome) are not retryable here.
func Retry(ctx context.Context, attempts int, fn func(context.Context) error) error {
	if attempts < 1 {
		return fmt.Errorf("attempts must be positive")
	}
	for n := 0; ; n++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := fn(ctx)
		var pgErr *pgconn.PgError
		if err == nil || n+1 >= attempts || !errors.As(err, &pgErr) || (pgErr.Code != "40001" && pgErr.Code != "40P01") {
			return err
		}
		timer := time.NewTimer(time.Duration(n+1) * 10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
