package main

import "fmt"

func get() string {
	fmt.Println("1")
	return ""
}

func handle(string) {
	fmt.Println("3")
}

func process() {
	defer handle(get())
	fmt.Println("2")
}

/*
 * 1 prints first because get() is called before defer
 * 2 prints second because process() is called
 * 3 prints third because handle() is deferred
 */
func main() {
	process()
}
