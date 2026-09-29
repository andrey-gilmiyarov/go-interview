package solution

import "context"

func Square(ctx context.Context, in <-chan int) <-chan int {
	out := make(chan int)
	if ctx.Err() != nil {
		close(out)
		return out
	}
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case value, ok := <-in:
				if !ok {
					return
				}
				select {
				case out <- value * value:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}
