package main

import "fmt"

func process1() {
	defer fmt.Println(3)
	defer fmt.Println(2)
	defer fmt.Println(1)
}

func process2() {
	defer func() {
		defer fmt.Println(4)
		defer fmt.Println(3)
	}()

	defer func() {
		defer fmt.Println(2)
		defer fmt.Println(1)
	}()
}

/* 
 * This program demonstrates the order of execution of deferred function calls in Go.
 * In the process1 function, three deferred calls to fmt.Println are made.
 * They will be executed in last-in-first-out (LIFO) order when the function returns,
 * resulting in the output "1", "2", "3".
 *
 * In the process2 function, there are two deferred anonymous functions.
 * Each of these functions contains its own deferred calls to fmt.Println.
 * When process2 returns, the outer deferred function is executed first,
 * which in turn executes its inner deferred calls in LIFO order,
 * resulting in the output "1", "2", "3", "4".
 */
func main() {
	process1()
	process2()
}
