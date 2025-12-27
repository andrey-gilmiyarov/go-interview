package channels

import (
	"fmt"
	"net/http"
	"sync"
)

func Pool() {
    urls := []string{
        "https://www.youtube.com/",
        "https://wikipedia.org/",
        "https://google.com/",
        "https://techcrunch.com/",
        "https://leetcode.com/",
    }

    numWorkers := 3
    jobs := make(chan string)
    results := make(chan int)

    // Producer
    go func() {
        for _, url := range urls {
            jobs <- url
        }
        close(jobs)
    }()

    var wg sync.WaitGroup

    // Workers
    for i := 1; i <= numWorkers; i++ {
        wg.Go(func() {
            defer wg.Done()
            for job := range jobs {
                fmt.Printf("Worker %d processing job %s\n", i, job)

                resp, err := http.Get(job)
                if err != nil {
                    fmt.Printf("Worker %d error: %v\n", i, err)
                    continue
                }

                // we only care about status code here
                results <- resp.StatusCode
                resp.Body.Close()
            }
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
