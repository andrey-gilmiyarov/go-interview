package solution

import "github.com/andreygilmiyarov/go-interview/practice/select-states/contract"

func Predict() contract.Answer {
	return contract.Answer{ReadyOutcomes: []string{"left", "right"}, NilBlocks: true, ClosedValue: 0, ClosedOK: false}
}
