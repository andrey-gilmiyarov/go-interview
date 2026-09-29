package example

import (
	"context"
	"errors"

	"github.com/andreygilmiyarov/go-interview/practice/first-success/solution"
)

func TaxiExample(ctx context.Context) (string, error) {
	jobs := []func(context.Context) (string, error){
		func(context.Context) (string, error) {
			return "", errors.New("service is unavailable")
		},
		func(context.Context) (string, error) {
			return "Villagemobil", nil
		},
	}
	return solution.First(ctx, jobs...)
}
