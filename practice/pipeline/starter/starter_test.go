//go:build exercise

package starter

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/pipeline/internal/checks"
)

func TestSquare(t *testing.T) { checks.Run(t, Square) }
