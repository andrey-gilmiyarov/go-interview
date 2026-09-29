//go:build exercise

package starter

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/fan-in/internal/checks"
)

func TestMerge(t *testing.T) { checks.Run(t, Merge) }
