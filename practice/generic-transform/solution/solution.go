package solution

func Transform[T any, R any](values []T, fn func(T) R) []R {
	if values == nil {
		return nil
	}

	result := make([]R, len(values))
	for index, value := range values {
		result[index] = fn(value)
	}
	return result
}
