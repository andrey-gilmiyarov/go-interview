package channels

import (
	"fmt"
	"sync"
)

func FanInOut() {
	data := []int{1, 2, 3, 4, 5}
	input := make(chan int, len(data))
	for _, d := range data {
		input <- d
	}
	close(input)

	// Fan-out: Launch multiple worker goroutines
	numWorkers := 3
	results := make(chan int, len(data))

	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for num := range input {
				// Simulate some processing
				result := num * 2
				results <- result
			}
		}()
	}

	// Fan-in: Aggregate results from workers
	go func() {
		wg.Wait()
		close(results)
	}()

	// Process aggregated results
	for result := range results {
		fmt.Println(result)
	}
}