//go:build exercise

package starter

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/labs/postgres/internal/checks"
)

func TestContract(t *testing.T) { checks.Queue(t, Claim, Ack) }
