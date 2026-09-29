package solution

import (
	"errors"

	"github.com/andreygilmiyarov/go-interview/practice/multi-error/contract"
)

var ErrNegative = contract.ErrNegative

type ItemError = contract.ItemError

func Validate(values []int) error {
	var problems []error
	for index, value := range values {
		if value < 0 {
			problems = append(problems, &contract.ItemError{Index: index, Value: value})
		}
	}
	return errors.Join(problems...)
}
