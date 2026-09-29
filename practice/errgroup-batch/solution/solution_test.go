package solution

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/errgroup-batch/internal/checks"
)

func TestRun(t *testing.T) { checks.Run(t, Run) }
