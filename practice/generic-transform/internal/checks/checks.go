package checks

import "testing"

func RunIntToString(t *testing.T, transform func([]int, func(int) string) []string) {
	t.Helper()
	t.Run("preserves order and nil input", func(t *testing.T) {
		got := transform([]int{9, 2, 7}, func(value int) string {
			return string(rune('0' + value))
		})
		want := []string{"9", "2", "7"}
		if len(got) != len(want) {
			t.Fatalf("result length = %d, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("result[%d] = %q, want %q", i, got[i], want[i])
			}
		}
		if got := transform(nil, func(value int) string { return "unused" }); got != nil {
			t.Fatalf("Transform(nil) = %#v, want nil", got)
		}
	})

	t.Run("keeps a non-nil empty result non-nil", func(t *testing.T) {
		input := make([]int, 0)
		got := transform(input, func(value int) string { return "unused" })
		if got == nil || len(got) != 0 {
			t.Fatalf("Transform(empty) = %#v, want non-nil empty slice", got)
		}
	})
}

func RunStringToInt(t *testing.T, transform func([]string, func(string) int) []int) {
	t.Helper()
	t.Run("supports a different output type", func(t *testing.T) {
		got := transform([]string{"π", "go", ""}, func(value string) int {
			return len([]rune(value))
		})
		want := []int{1, 2, 0}
		if len(got) != len(want) {
			t.Fatalf("result length = %d, want %d", len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("result[%d] = %d, want %d", i, got[i], want[i])
			}
		}
	})
}
