package solution

import "github.com/andreygilmiyarov/go-interview/practice/range-mutation/contract"

func Predict() contract.Answer {
	return contract.Answer{Seen: []int{1, 2, 3}, Final: []int{1, 2, 3, 1, 2, 3}}
}
