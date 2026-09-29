//go:build exercise

package starter

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/allocations/internal/checks"
)

func TestJoin(t *testing.T) { checks.Run(t, Join) }

func BenchmarkJoin(b *testing.B) { checks.BenchmarkJoin(b, Join) }
