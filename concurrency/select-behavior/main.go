package main

import (
	"fmt"
	"time"
)


func main() {
	messageCh := make(chan int, 10)
	disconnectCh := make(chan struct{})

	go listing1(messageCh, disconnectCh)
	//go listing2(messageCh, disconnectCh)

	for i := range 10 {
		messageCh <- i
	}
	disconnectCh <- struct{}{}
	time.Sleep(10 * time.Millisecond)
}

/*
	Timeline: why you can exit early

	1. The producer goroutine runs:
	 - It does messageCh <- 0, messageCh <- 1, ..., up to messageCh <- 9.
	 - Because messageCh has capacity 10, all 10 sends complete immediately, without waiting for the consumer.
	 - Now messageCh’s buffer = [0,1,2,3,4,5,6,7,8,9].

	2. After the loop, the producer executes: `disconnectCh <- struct{}{}`
	This is an unbuffered send, so it blocks until someone receives from disconnectCh.

	3. Meanwhile, your listing1 goroutine is running and repeatedly doing the select.
	At this moment:
	- messageCh is non-empty → receive from messageCh is ready.
	- The producer is blocked trying to send on disconnectCh → receive from disconnectCh is also ready.
	So in each iteration of select, both cases are ready.

	4. Per the Go spec: if multiple cases in a select are ready, one of them is chosen at random (pseudo-random, but essentially you should treat it as arbitrary).

	That means:
	- iteration 1: select might choose case v := <-messageCh → you print 0.
	- iteration 2: now both are still ready (messageCh has [1..9], disconnectCh has pending send).
	select might choose either: message case again → print 1 or disconnect case → print "disconnection, return" and return.

	As soon as disconnect case wins even once, the function returns and stops reading from messageCh, leaving some numbers unprinted.
 */
func listing1(messageCh <-chan int, disconnectCh chan struct{}) {
	for {
		select {
		case v := <-messageCh:
			fmt.Println(v)
		case <-disconnectCh:
			fmt.Println("disconnection, return")
			return
		}
	}
}

/*
	This is a drain loop:

	- case v := <-messageCh:
	Runs as long as there is any value buffered in messageCh (receive is ready).

	- default:
	Runs only when messageCh has no value ready (i.e. buffer is empty and no sender ready).

	Given that:
	1. All 10 values were sent before disconnectCh <- struct{}{}.
	2. No more values will be sent after that.
	3. You only enter this inner loop after reading from disconnectCh.

	Then:
	- When you first enter the inner loop, messageCh contains all remaining values that weren’t already printed by the outer loop.
	- The inner select will repeatedly pick the case v := <-messageCh branch until the channel is empty.
	- Only when messageCh is fully drained does the receive case become not-ready, so the default.

	So regardless of whether the outer loop printed 0 values, 3 values, or all 10 values before hitting the disconnect case, 
	the inner loop will print all remaining values and only then print "disconnection, return".
 */
func listing2(messageCh <-chan int, disconnectCh chan struct{}) {
	for {
		select {
		case v := <-messageCh:
			fmt.Println(v)
		case <-disconnectCh:
			for {
				select {
				case v := <-messageCh:
					fmt.Println(v)
				default:
					fmt.Println("disconnection, return")
					return
				}
			}
		}
	}
}