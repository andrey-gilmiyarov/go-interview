package solution

import "github.com/andreygilmiyarov/go-interview/practice/slice-alias/contract"

func Predict() contract.Answer {
	return contract.Answer{Shared: []int{1, 2, 9}, Detached: []int{1, 2, 9}}
}
