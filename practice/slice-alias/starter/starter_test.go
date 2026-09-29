//go:build exercise

package starter

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/slice-alias/internal/checks"
)

func TestPredict(t *testing.T) { checks.Run(t, Predict) }
