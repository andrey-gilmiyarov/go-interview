package solution

import "github.com/andreygilmiyarov/go-interview/practice/defer-order/contract"

func Predict() contract.Answer {
	return contract.Answer{Events: []string{"2", "1"}}
}
