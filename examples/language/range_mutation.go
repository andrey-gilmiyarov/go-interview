package language

type RangeVisit struct {
	Index int
	Value string
}

type ComplicatedRangeResult struct {
	Seen  []RangeVisit
	Final []string
}

// ComplicatedRangeMutation retains the original write-append-write trace.
func ComplicatedRangeMutation() ComplicatedRangeResult {
	values := []string{"A", "M", "C"}
	result := ComplicatedRangeResult{Seen: make([]RangeVisit, 0, len(values))}
	for i, value := range values {
		result.Seen = append(result.Seen, RangeVisit{Index: i, Value: value})
		values[i+1] = "M"
		values = append(values, "Z")
		values[i+1] = "Z"
	}
	result.Final = append([]string(nil), values...)
	return result
}
