package selectbehavior

import (
	"fmt"
	"time"
)

/*
Key issue: When multiple cases in a select are ready, Go randomly picks one to execute.
In the Run function:
10 messages are sent to messageCh (which has a buffer of 10)
Then a disconnect signal is sent to disconnectCh
At this point, both channels have data ready:
messageCh has buffered messages (0-9)
disconnectCh has the disconnect signal
Each time through the loop, the select randomly chooses between:
Reading a message from messageCh
Reading from disconnectCh and returning

When the disconnect signal is received, GoodListing:
Doesn't immediately return
Enters a draining loop that processes all remaining messages
Uses a default case that only executes when messageCh is empty
This ensures all 10 messages are printed before the function returns, giving you predictable, complete output.
This pattern is called graceful shutdown - ensuring all pending work is completed before shutting down, rather than abruptly stopping and potentially losing data.
*/

func Run(listing func(messageCh <-chan int, disconnectCh chan struct{})) {
	messageCh := make(chan int, 10)
	disconnectCh := make(chan struct{})

	go listing(messageCh, disconnectCh)

	for i := 0; i < 10; i++ {
		messageCh <- i
	}
	disconnectCh <- struct{}{}
	time.Sleep(10 * time.Millisecond)
}

func BadListing(messageCh <-chan int, disconnectCh chan struct{}) {
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

func GoodListing(messageCh <-chan int, disconnectCh chan struct{}) {
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