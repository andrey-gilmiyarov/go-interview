package checks

import (
	"reflect"
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/defer-order/contract"
)

func Run(t *testing.T, predict func() contract.Answer) {
	t.Helper()
	want := contract.Answer{Events: []string{"2", "1"}}
	if got := predict(); !reflect.DeepEqual(got, want) {
		t.Errorf("Predict() = %#v, want %#v", got, want)
	}
	if got := contract.Observe().Answer; !reflect.DeepEqual(got, want) {
		t.Errorf("canonical example = %#v, want %#v", got, want)
	}
}
