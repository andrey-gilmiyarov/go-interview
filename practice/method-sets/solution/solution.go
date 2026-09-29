package solution

import "github.com/andreygilmiyarov/go-interview/practice/method-sets/contract"

func Predict() contract.Answer {
	return contract.Answer{ValueImplements: false, PointerImplements: true, EmbeddedValueImplements: true}
}
