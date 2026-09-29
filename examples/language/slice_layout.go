package language

type arrayOfStruct struct {
	a int64
	b int64
}

func sumArrayOfStructs(values []arrayOfStruct) int64 {
	var total int64
	for i := range values {
		total += values[i].a
	}
	return total
}

type structOfSlices struct {
	a []int64
	b []int64
}

func sumStructOfSlices(values structOfSlices) int64 {
	var total int64
	for i := range values.a {
		total += values.a[i]
	}
	return total
}
