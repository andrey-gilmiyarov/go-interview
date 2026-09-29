package checks

import (
	"reflect"
	"strings"
	"testing"
)

func Run(t *testing.T, join func([]string) string) {
	t.Helper()
	tests := []struct {
		name  string
		parts []string
		want  string
	}{
		{"nil", nil, ""},
		{"empty", []string{}, ""},
		{"single empty", []string{""}, ""},
		{"empty values", []string{"", "", ""}, ",,"},
		{"unicode", []string{"Привет", "λ", "猫"}, "Привет,λ,猫"},
		{"mixed values", []string{"Go", "", "2026"}, "Go,,2026"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := append([]string(nil), test.parts...)
			if test.parts != nil && before == nil {
				before = []string{}
			}
			if got := join(test.parts); got != test.want {
				t.Fatalf("Join(%q) = %q, want %q", test.parts, got, test.want)
			}
			if !reflect.DeepEqual(test.parts, before) {
				t.Fatalf("Join mutated input: got %#v, want %#v", test.parts, before)
			}
			if got, want := join(test.parts), strings.Join(test.parts, ","); got != want {
				t.Fatalf("Join semantics differ from strings.Join: %q != %q", got, want)
			}
		})
	}
}

const benchmarkPartCount = 128

var benchmarkSink string

func BenchmarkJoin(b *testing.B, join func([]string) string) {
	parts := makeBenchmarkParts()
	b.Run("implementation", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkSink = join(parts)
		}
	})
	b.Run("naive-concatenation", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			benchmarkSink = naiveJoin(parts)
		}
	})
}

func makeBenchmarkParts() []string {
	parts := make([]string, benchmarkPartCount)
	for i := range parts {
		parts[i] = strings.Repeat("x", 24)
	}
	return parts
}

func naiveJoin(parts []string) string {
	result := ""
	for i, part := range parts {
		if i > 0 {
			result += ","
		}
		result += part
	}
	return result
}
