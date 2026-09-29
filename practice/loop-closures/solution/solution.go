package solution

import "github.com/andreygilmiyarov/go-interview/practice/loop-closures/contract"

func Predict() contract.Answer {
	return contract.Answer{Declared: []int{0, 1, 2}, Reused: []int{3, 3, 3}}
}
