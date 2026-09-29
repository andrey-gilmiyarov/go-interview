package contract

import (
	"errors"
	"fmt"
)

var ErrNegative = errors.New("negative value")

type ItemError struct {
	Index int
	Value int
}

func (e *ItemError) Error() string {
	return fmt.Sprintf("value %d at index %d: %v", e.Value, e.Index, ErrNegative)
}

func (e *ItemError) Unwrap() error { return ErrNegative }
