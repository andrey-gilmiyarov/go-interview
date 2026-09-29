package example

import (
	"strconv"

	"github.com/andreygilmiyarov/go-interview/practice/generic-transform/starter"
)

func TransformExample() []string {
	return starter.Transform([]int{4, 7, 9}, strconv.Itoa)
}
