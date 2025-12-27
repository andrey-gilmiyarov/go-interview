package errgroup

import (
	"context"
	"fmt"
	"golang.org/x/sync/errgroup"
	"test/goroutine"
)

var URLList = []string{
	"https://www.youtube.com/",
	"https://en.wikipedia.org/",
	"https://google.com/",
	"https://ya.ru/",
	"https://habr.com/",
	"https://leetcode.com",
}

func GetRespFromURLList() error {
	ctx := context.Background()

	var g errgroup.Group
	for _, url := range URLList {
		url := url // Create a local copy for the closure
		g.Go(func() error {
			resp, err := goroutine.GetResp(ctx, url)
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

	fmt.Println("All URLs processed successfully")
	return nil
}