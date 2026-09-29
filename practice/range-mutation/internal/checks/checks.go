package checks

import (
	"reflect"
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/range-mutation/contract"
)

func Run(t *testing.T, predict func() contract.Answer) {
	t.Helper()
	want := contract.Answer{Seen: []int{1, 2, 3}, Final: []int{1, 2, 3, 1, 2, 3}}
	if got := predict(); !reflect.DeepEqual(got, want) {
		t.Errorf("Predict() = %#v, want %#v", got, want)
	}
	if got := contract.Observe().Answer; !reflect.DeepEqual(got, want) {
		t.Errorf("canonical example = %#v, want %#v", got, want)
	}
}
