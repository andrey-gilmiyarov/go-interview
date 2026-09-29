package checks

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/map-iteration/contract"
)

func Run(t *testing.T, predict func() contract.Answer) {
	t.Helper()
	want := contract.Answer{OrderGuaranteed: false, MustVisitInserted: false, MustVisitDeletedBeforeReached: false}
	if got := predict(); got != want {
		t.Errorf("Predict() = %#v, want %#v", got, want)
	}

	observed := contract.Observe()
	if observed.Answer != want {
		t.Errorf("canonical answer = %#v, want %#v", observed.Answer, want)
	}
	initial := make(map[string]bool, len(observed.InitialKeys))
	for _, key := range observed.InitialKeys {
		if initial[key] {
			t.Errorf("canonical initial key %q repeated", key)
		}
		initial[key] = true
	}
	if len(initial) != len(observed.InitialVisits) {
		t.Errorf("canonical visits = %#v for initial keys %#v", observed.InitialVisits, initial)
	}
	for key, count := range observed.InitialVisits {
		if !initial[key] {
			t.Errorf("canonical iteration returned unexpected initial key %q", key)
		}
		if count != 1 {
			t.Errorf("canonical key %q visited %d times, want once", key, count)
		}
	}
	for key := range initial {
		if observed.InitialVisits[key] != 1 {
			t.Errorf("canonical initial key %q was not visited once", key)
		}
	}
	if observed.DeletedBeforeReached && observed.DeletedKeyWasVisited {
		t.Error("canonical range visited a key after deleting it before its turn")
	}
	// Either insertion observation is permitted by the language specification.
	_ = observed.InsertedKeyWasVisited
}
