package solution

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/nil-empty-json/internal/checks"
)

func TestPredict(t *testing.T) { checks.Run(t, Predict) }
