package main

import (
	"fmt"
	"unsafe"
)

/*
 * A slice value is a tiny header: {Data pointer, Len, Cap}.
 * On 64-bit Go, unsafe.Sizeof(slice) is 24 bytes (3 machine words).
 * That size is the same no matter what the slice contains; it’s the header size, not the backing array.
 */
func main() {
	var data []string
	fmt.Println("This declares a nil slice (the zero value)")
	fmt.Println("var data []string:")
	fmt.Printf("\tempty=%t nil=%t size=%d data=%p\n\n", len(data) == 0, data == nil, unsafe.Sizeof(data), unsafe.SliceData(data))

	data = []string(nil)
	fmt.Println("Explicitly assigns the nil slice value")
	fmt.Println("data = []string(nil):")
	fmt.Printf("\tempty=%t nil=%t size=%d data=%p\n\n", len(data) == 0, data == nil, unsafe.Sizeof(data), unsafe.SliceData(data))

	data = []string{}
	fmt.Println("This is an empty but non-nil slice literal")
	fmt.Println("data = []string{}:")
	fmt.Printf("\tempty=%t nil=%t size=%d data=%p\n\n", len(data) == 0, data == nil, unsafe.Sizeof(data), unsafe.SliceData(data))

	data = make([]string, 0)
	fmt.Println("Also produces an empty but non-nil slice")
	fmt.Println("data = make([]string, 0):")
	fmt.Printf("\tempty=%t nil=%t size=%d data=%p\n\n", len(data) == 0, data == nil, unsafe.Sizeof(data), unsafe.SliceData(data))

	empty := struct{}{}
	fmt.Println("Empty struct address:", unsafe.Pointer(&empty))
}
