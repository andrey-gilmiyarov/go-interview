//go:build exercise

package starter

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/multi-error/internal/checks"
)

func TestValidate(t *testing.T) { checks.Run(t, Validate) }
