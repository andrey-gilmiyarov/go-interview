package starter

import "github.com/andreygilmiyarov/go-interview/practice/multi-error/contract"

var ErrNegative = contract.ErrNegative

type ItemError = contract.ItemError

func Validate(values []int) error {
	panic("TODO")
}
