package channels

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func FindTaxi() {
	var (
			resultCh    = make(chan string, 1)
			ctx, cancel = context.WithCancel(context.Background())
			services    = []string{"Super", "Villagemobil", "Sett Taxi", "Index Go"}
			wg          sync.WaitGroup
		)
		
		defer cancel()
	
		wg.Add(len(services))
		for i := range services {
			svc := services[i]
	
			go func() {
				defer wg.Done()
				requestRide(ctx, svc, resultCh)
			}()
		}
	
		go func() {
			fmt.Println("start waiting")
			wg.Wait() // Wait for all goroutines to finish
			fmt.Println("all requests done, cancelling the context")
			cancel() // Cancel the context after all requests are done
			close(resultCh)
		}()
	
		var i int
		for {
			select {
				case <-ctx.Done():
					fmt.Println("context is cancelled")
					return
				case <-resultCh:
					i++
					fmt.Printf("get winner №%d - %s", i, <-resultCh)
			}
		}
}

func requestRide(ctx context.Context, serviceName string, resultCh chan<- string) {
	time.Sleep(3 * time.Second)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("stopped the search in %q (%v)", serviceName, ctx.Err())
			return
		default:
			if rand.Float64() > 0.75 {
				fmt.Printf("found service %s for request", serviceName)
				resultCh <- serviceName
				return
			}

			continue
		}
	}
}
