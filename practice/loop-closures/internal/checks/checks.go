package checks

import (
	"reflect"
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/loop-closures/contract"
)

func Run(t *testing.T, predict func() contract.Answer) {
	t.Helper()
	want := contract.Answer{Declared: []int{0, 1, 2}, Reused: []int{3, 3, 3}}
	if got := predict(); !reflect.DeepEqual(got, want) {
		t.Errorf("Predict() = %#v, want %#v", got, want)
	}
	if got := contract.Observe().Answer; !reflect.DeepEqual(got, want) {
		t.Errorf("canonical example = %#v, want %#v", got, want)
	}
}
