package goroutine

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
)

func main() {
	var urls = []string{
		"https://www.youtube.com/",
		"https://en.wikipedia.org/",
		"https://google.com/",
	}

	var g errgroup.Group

	ctx := context.Background()

	for _, url := range urls {
		url := url // Create a local copy for the closure
		g.Go(func() error {
			resp, err := GetResp(ctx, url)
			if err != nil {
				fmt.Printf("Get error %s when trying to fetch %s url\n", err, url)
				return err
			}
			fmt.Printf("URL %s with status %s\n", url, resp.Status)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
