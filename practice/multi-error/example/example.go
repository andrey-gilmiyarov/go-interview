package example

import "github.com/andreygilmiyarov/go-interview/practice/multi-error/starter"

func ValidateBatch(values []int) error {
	return starter.Validate(values)
}
