package solution

import "github.com/andreygilmiyarov/go-interview/practice/named-return/contract"

func Predict() contract.Answer {
	return contract.Answer{Result: 5}
}
