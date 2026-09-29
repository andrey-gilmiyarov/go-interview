package contract

func example() (actual string, leftReady, rightReady bool, nilSelected bool, closedValue int, closedOK bool) {
	left := make(chan struct{}, 1)
	right := make(chan struct{}, 1)
	left <- struct{}{}
	right <- struct{}{}
	leftReady = len(left) > 0
	rightReady = len(right) > 0

	select {
	case <-left:
		actual = "left"
	case <-right:
		actual = "right"
	}

	var nilChannel <-chan int
	select {
	case <-nilChannel:
		nilSelected = true
	default:
	}

	closed := make(chan int)
	close(closed)
	closedValue, closedOK = <-closed
	return
}
