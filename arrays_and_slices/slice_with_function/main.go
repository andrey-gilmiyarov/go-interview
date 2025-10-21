package main

import "fmt"

func main() {
	data := make([]int, 0, 5) // len = 0, cap = 5
	data = append(data, []int{1, 2, 3, 4}...) // len = 4, cap = 5

	fmt.Printf("original slice: %v; len: %d; cap: %d\n", data, len(data), cap(data))
	process1(data)
	fmt.Printf("after process1: %v; len: %d; cap: %d\n\n", data, len(data), cap(data))
	
	process2(data)
	// slice's headeds didn't change so len and array slice stay the same
	fmt.Printf("after process2: %v; len: %d; cap: %d\n\n", data, len(data), cap(data))
	
	// append to the same array and we see it after reslice
	reslicedData := data[:5]
	fmt.Printf("resliced data:  %v; len: %d; cap: %d\n", reslicedData, len(reslicedData), cap(reslicedData))
}

func process1(data []int) {
	data[2] = 5
}

func process2(data []int) {
	data = append(data, 6)
	fmt.Println("in process2 copy of data:", data, "len:", len(data), "cap:", cap(data))
}
