//go:build exercise

package starter

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/worker-pool/internal/checks"
)

func TestMap(t *testing.T) { checks.Run(t, Map) }
