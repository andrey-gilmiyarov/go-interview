package solution

import "github.com/andreygilmiyarov/go-interview/practice/typed-nil/contract"

func Predict() contract.Answer {
	return contract.Answer{ErrorIsNil: false, PointerIsNil: true}
}
