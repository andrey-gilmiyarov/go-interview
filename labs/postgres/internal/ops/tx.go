package ops

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrStock = errors.New("insufficient stock")
var ErrConflict = errors.New("idempotency key reused with different payload")

func Rollback(tx pgx.Tx) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = tx.Rollback(ctx)
}
