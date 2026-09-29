package language

// StopWithoutDraining retains the early-return branch from the original exercise.
// If messages and disconnect are ready together, select may stop with messages queued.
func StopWithoutDraining(messages <-chan int, disconnect <-chan struct{}) []int {
	var received []int
	for {
		select {
		case value, ok := <-messages:
			if !ok {
				return received
			}
			received = append(received, value)
		case <-disconnect:
			return received
		}
	}
}

func DrainOnDisconnect(messages <-chan int, disconnect <-chan struct{}) []int {
	var received []int
	for {
		select {
		case value, ok := <-messages:
			if !ok {
				return received
			}
			received = append(received, value)
		case <-disconnect:
			for {
				select {
				case value, ok := <-messages:
					if !ok {
						return received
					}
					received = append(received, value)
				default:
					return received
				}
			}
		}
	}
}

// SelectDrainExample creates a ready stop signal and a buffered, closed input.
// It needs no goroutine or timing assumption and always returns the queued values.
func SelectDrainExample() []int {
	messages := make(chan int, 10)
	disconnect := make(chan struct{}, 1)
	for value := 0; value < 10; value++ {
		messages <- value
	}
	close(messages)
	disconnect <- struct{}{}
	return DrainOnDisconnect(messages, disconnect)
}
