package language

func DeferArgumentTrace() (events []int) {
	defer recordDeferredArgument(&events, evaluateDeferredArgument(&events))
	events = append(events, 2)
	return
}

func evaluateDeferredArgument(events *[]int) string {
	*events = append(*events, 1)
	return ""
}

func recordDeferredArgument(events *[]int, _ string) {
	*events = append(*events, 3)
}

type valueReceiver struct{ value int }

func (receiver valueReceiver) record(events *[]int) {
	*events = append(*events, receiver.value)
}

type pointerReceiver struct{ value int }

func (receiver *pointerReceiver) record(events *[]int) {
	*events = append(*events, receiver.value)
}

func DeferReceiverTrace() (events []int) {
	value := valueReceiver{}
	defer value.record(&events)

	pointer := pointerReceiver{}
	defer pointer.record(&events)

	value.value = 100
	pointer.value = 200
	return
}

func DeferredOrderTraces() (first, nested []int) {
	first = processFlatDefers()
	nested = processNestedDefers()
	return first, nested
}

func processFlatDefers() (events []int) {
	defer func() { events = append(events, 3) }()
	defer func() { events = append(events, 2) }()
	defer func() { events = append(events, 1) }()
	return
}

func processNestedDefers() (events []int) {
	defer func() {
		defer func() { events = append(events, 4) }()
		defer func() { events = append(events, 3) }()
	}()
	defer func() {
		defer func() { events = append(events, 2) }()
		defer func() { events = append(events, 1) }()
	}()
	return
}
