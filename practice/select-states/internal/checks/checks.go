package checks

import (
	"testing"

	"github.com/andreygilmiyarov/go-interview/practice/select-states/contract"
)

func sameSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	values := make(map[string]int, len(left))
	for _, value := range left {
		values[value]++
	}
	for _, value := range right {
		values[value]--
	}
	for _, count := range values {
		if count != 0 {
			return false
		}
	}
	return true
}

func equalAnswer(left, right contract.Answer) bool {
	return sameSet(left.ReadyOutcomes, right.ReadyOutcomes) &&
		left.NilBlocks == right.NilBlocks &&
		left.ClosedValue == right.ClosedValue &&
		left.ClosedOK == right.ClosedOK
}

func Run(t *testing.T, predict func() contract.Answer) {
	t.Helper()
	want := contract.Answer{ReadyOutcomes: []string{"left", "right"}, NilBlocks: true, ClosedValue: 0, ClosedOK: false}
	if got := predict(); !equalAnswer(got, want) {
		t.Errorf("Predict() = %#v, want set-equivalent to %#v", got, want)
	}
	observed := contract.Observe()
	if !equalAnswer(observed.Answer, want) {
		t.Errorf("canonical answer = %#v, want set-equivalent to %#v", observed.Answer, want)
	}
	if !observed.LeftWasReady || !observed.RightWasReady {
		t.Errorf("canonical select did not start with both ready channels: %#v", observed)
	}
	if observed.ActualReadyOutcome != "left" && observed.ActualReadyOutcome != "right" {
		t.Errorf("select chose invalid outcome %q", observed.ActualReadyOutcome)
	}
	if observed.NilCaseWasSelected {
		t.Error("nil-channel case was selected while a default case was available")
	}
}
