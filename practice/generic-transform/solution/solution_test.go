package solution

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/generic-transform/internal/checks"
)

func TestTransform(t *testing.T) {
	checks.RunIntToString(t, Transform[int, string])
	checks.RunStringToInt(t, Transform[string, int])
}
