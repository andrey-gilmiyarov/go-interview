package solution

import "github.com/andreygilmiyarov/go-interview/practice/nil-empty-json/contract"

func Predict() contract.Answer {
	return contract.Answer{NilEqual: true, EmptyEqual: false, NilJSON: "null", EmptyJSON: "[]"}
}
