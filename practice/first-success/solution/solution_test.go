package solution

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/first-success/internal/checks"
)

func TestFirst(t *testing.T) { checks.Run(t, ErrNoJobs, First) }
