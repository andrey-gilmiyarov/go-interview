package solution

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/http-middleware/internal/checks"
)

func TestHandler(t *testing.T) { checks.Run(t, Handler) }
