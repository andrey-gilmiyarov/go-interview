//go:build integration

package solution

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/labs/postgres/internal/checks"
)

func TestContract(t *testing.T) { checks.Pool(t, ReadStocks) }
