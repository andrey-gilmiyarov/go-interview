package checks

import (
	"reflect"
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/nil-empty-json/contract"
)

func Run(t *testing.T, predict func() contract.Answer) {
	t.Helper()
	want := contract.Answer{NilEqual: true, EmptyEqual: false, NilJSON: "null", EmptyJSON: "[]"}
	if got := predict(); !reflect.DeepEqual(got, want) {
		t.Errorf("Predict() = %#v, want %#v", got, want)
	}
	if got := contract.Observe().Answer; !reflect.DeepEqual(got, want) {
		t.Errorf("canonical example = %#v, want %#v", got, want)
	}
}
