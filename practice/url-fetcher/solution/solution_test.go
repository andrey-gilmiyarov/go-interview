package solution

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/url-fetcher/internal/checks"
)

func TestFetchAll(t *testing.T) { checks.Run(t, FetchAll) }
