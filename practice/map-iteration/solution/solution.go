package solution

import "github.com/andreygilmiyarov/go-interview/practice/map-iteration/contract"

func Predict() contract.Answer {
	return contract.Answer{OrderGuaranteed: false, MustVisitInserted: false, MustVisitDeletedBeforeReached: false}
}
