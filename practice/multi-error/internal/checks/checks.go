package checks

import (
	"errors"
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/multi-error/contract"
)

func Run(t *testing.T, validate func([]int) error) {
	t.Helper()

	t.Run("valid values return a true nil", func(t *testing.T) {
		if err := validate([]int{0, 2, 19}); err != nil {
			t.Fatalf("Validate(valid) = %#v, want nil", err)
		}
		if err := validate(nil); err != nil {
			t.Fatalf("Validate(nil) = %#v, want nil", err)
		}
	})

	t.Run("joins every indexed negative value", func(t *testing.T) {
		err := validate([]int{4, -3, 0, -8})
		if err == nil {
			t.Fatal("Validate accepted negative values")
		}
		if !errors.Is(err, contract.ErrNegative) {
			t.Fatalf("errors.Is(error, ErrNegative) = false for %v", err)
		}
		joined, ok := err.(interface{ Unwrap() []error })
		if !ok || len(joined.Unwrap()) != 2 {
			t.Fatalf("joined error children = %v, want two", err)
		}
		want := map[int]int{1: -3, 3: -8}
		for _, child := range joined.Unwrap() {
			var item *contract.ItemError
			if !errors.As(child, &item) {
				t.Fatalf("errors.As(child, *ItemError) = false for %v", child)
			}
			if value, ok := want[item.Index]; !ok || value != item.Value {
				t.Errorf("unexpected ItemError = %#v", item)
			}
			delete(want, item.Index)
		}
		if len(want) != 0 {
			t.Errorf("missing indexed errors: %#v", want)
		}
	})
}
