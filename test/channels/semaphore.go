package channels

import (
	"fmt"
	"net/http"
	"sync"
)

func Semaphore() {
	urls := []string{
        "https://www.youtube.com/",
        "https://wikipedia.org/",
        "https://google.com/",
        "https://techcrunch.com/",
        "https://leetcode.com/",
    }

    numWorkers := 3

	sem := make(chan struct{}, numWorkers)
	results := make(chan int)
	var wg sync.WaitGroup

	for _, url := range urls {
        wg.Go(func() {
			defer func() { <-sem }() // release slot

			sem <- struct{}{} // acquire slot

			fmt.Printf("Worker processing url %s\n", url)
            resp, _ := http.Get(url)

            // we only care about status code here
            results <- resp.StatusCode
            resp.Body.Close()
		})
	}

	// Closer for results
    go func() {
        wg.Wait()
        close(results)
    }()

    // Consumer
    for status := range results {
        fmt.Printf("Result: %d\n", status)
    }
}