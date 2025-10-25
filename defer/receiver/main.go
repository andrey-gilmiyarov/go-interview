package main

import "fmt"

type data1 struct {
	value int
}

func (d data1) print() {
	fmt.Println("data1", d.value)
}

type data2 struct {
	value int
}

func (d *data2) print() {
	fmt.Println("data2", d.value)
}

/* 
 * When using defer with methods, the receiver type (value or pointer)
 * determines how the method is called at the time of defer.
 * 
 * In this example:
 * - data1 has a value receiver, so the value is captured at the time of defer.
 * - data2 has a pointer receiver, so the current value is accessed at the time of execution.
 */
func main() {
	d1 := data1{}
	defer d1.print()

	d2 := data2{}
	defer d2.print()

	d1.value = 100
	d2.value = 200
}
