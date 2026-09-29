//go:build exercise

package starter

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/named-return/internal/checks"
)

func TestPredict(t *testing.T) { checks.Run(t, Predict) }
