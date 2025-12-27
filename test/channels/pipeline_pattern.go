package channels

import (
	"fmt"
	"sync"
)

func Pipe() {
	done := make(chan struct{}, 2)
	defer close(done)

	in := gen(2, 3)

	c1 := sq(done, in)
	c2 := sq(done, in)

	for n := range merge(done, c1, c2) {
		fmt.Println(n)
	}
}

func gen(nums ...int) <-chan int {
	out := make(chan int, len(nums))
	defer close(out)
	
	for _, n := range nums {
		out <- n
	}
	
	return out
}

func sq(done <-chan struct{}, in <-chan int) <-chan int {
	out := make(chan int)
	defer close(out)
	
	go func() {
		for n := range in {
			select {
			case out <- n * n:
				fmt.Println("sq num")
			case <-done:
				return
			}
		}
	}()
	
	return out
}

func merge(done <-chan struct{}, cs ...<-chan int) <-chan int {
	var wg sync.WaitGroup
	out := make(chan int)

	wg.Add(len(cs))
	for _, c := range cs {
		go func(c <-chan int) {
			defer wg.Done()
			for n := range c {
				select {
				case out <- n:
					fmt.Printf("recieved new value to out: %d\n", n)
				case <-done:
					fmt.Print("done")
					return
				}
			}
		}(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()
	
	return out
}
