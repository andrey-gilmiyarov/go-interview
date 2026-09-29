//go:build exercise

package starter

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/url-fetcher/internal/checks"
)

func TestFetchAll(t *testing.T) { checks.Run(t, FetchAll) }
