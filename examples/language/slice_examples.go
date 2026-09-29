package language

type SliceParameterSnapshot struct {
	AfterMutation   []int
	AfterAppendCall []int
	Resliced        []int
}

func SliceParameterExample() SliceParameterSnapshot {
	data := make([]int, 0, 5)
	data = append(data, 1, 2, 3, 4)
	processSliceElement(data)
	afterMutation := append([]int(nil), data...)
	appendInsideCallee(data)
	afterAppendCall := append([]int(nil), data...)
	resliced := append([]int(nil), data[:5]...)
	return SliceParameterSnapshot{
		AfterMutation: afterMutation, AfterAppendCall: afterAppendCall, Resliced: resliced,
	}
}

func processSliceElement(data []int) {
	data[2] = 5
}

func appendInsideCallee(data []int) {
	data = append(data, 6)
}

type AppendScenario struct {
	AfterCall []int
	Resliced  []int
}

func BadAppendScenarios() []AppendScenario {
	withCapacity := make([]int, 1, 4)
	withCapacity[0] = 1
	appendTwo(withCapacity)
	withCapacityResult := AppendScenario{
		AfterCall: append([]int(nil), withCapacity...),
		Resliced:  append([]int(nil), withCapacity[:3]...),
	}

	withoutCapacity := []int{1}
	appendTwo(withoutCapacity)
	withoutCapacityResult := AppendScenario{
		AfterCall: append([]int(nil), withoutCapacity...),
		Resliced:  append([]int(nil), withoutCapacity[:cap(withoutCapacity)]...),
	}
	return []AppendScenario{withCapacityResult, withoutCapacityResult}
}

func appendTwo(values []int) {
	values = append(values, 2, 3)
}

type OverlapAppendResult struct {
	Original [4]int
	Result   []int
}

func RepeatedOverlappingAppend() OverlapAppendResult {
	original := [...]int{0, 1, 2, 3}
	x := original[:1]
	y := original[2:]
	x = append(x, y...)
	x = append(x, y...)
	return OverlapAppendResult{Original: original, Result: append([]int(nil), x...)}
}
