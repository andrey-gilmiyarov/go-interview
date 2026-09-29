package language

import (
	"runtime"
	"testing"
)

const layoutSampleCount = 1_000_000

var layoutBenchmarkSink int64

func BenchmarkSumArrayOfStructs(b *testing.B) {
	values := make([]arrayOfStruct, layoutSampleCount)
	for i := range values {
		values[i] = arrayOfStruct{a: int64(i), b: int64(i)}
	}
	var sum int64
	for b.Loop() {
		sum = sumArrayOfStructs(values)
	}
	layoutBenchmarkSink = sum
	runtime.KeepAlive(values)
}

func BenchmarkSumStructOfSlices(b *testing.B) {
	values := structOfSlices{
		a: make([]int64, layoutSampleCount),
		b: make([]int64, layoutSampleCount),
	}
	for i := range values.a {
		values.a[i] = int64(i)
		values.b[i] = int64(i)
	}
	var sum int64
	for b.Loop() {
		sum = sumStructOfSlices(values)
	}
	layoutBenchmarkSink = sum
	runtime.KeepAlive(values.b)
}
