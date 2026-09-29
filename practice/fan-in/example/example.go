package example

import (
	"context"

	"github.com/andreygilmiyarov/go-interview/practice/fan-in/solution"
)

func FanInExample(ctx context.Context) []int {
	left := make(chan int, 2)
	left <- 1
	left <- 3
	close(left)

	right := make(chan int, 2)
	right <- 2
	right <- 4
	close(right)

	var values []int
	for value := range solution.Merge(ctx, left, right) {
		values = append(values, value)
	}
	return values
}
