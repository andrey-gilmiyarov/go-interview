package main

import (
	"fmt"
	"os"
)

type MyError struct{}

func (e *MyError) Error() string {
	return "error"
}

func ReadFile(filename string) error {
	var err *os.PathError
	if filename == "" {
		return err
	}

	// reading...
	return err
}

/*
 * In Go, an interface value is considered nil only if both its type and value are nil.
 * This means that if you have a nil pointer of a concrete type assigned to an interface,
 * the interface itself is not nil because it holds a type (the concrete type) even though the value is nil.
 *
 * In the first example, we have a nil pointer of type *MyError assigned to an error interface.
 * The interface holds the type *MyError, so it is not nil.
 * In the second example, ReadFile returns a nil pointer of type *os.PathError assigned to an error interface.
 * Again, the interface holds the type *os.PathError, so it is not nil.
 */
func main() {
	// Example 1
	var pointer *MyError = nil
	var err error = pointer
	fmt.Println("nil:", err == nil) // inside err (itab not nil, data nil)

	// Example 2
	err = ReadFile("")
	if err != nil {
		fmt.Println("error")
	} else {
		fmt.Println("nil")
	}

	fmt.Println("value of err: ", err)
	fmt.Printf("type of err: %T\n", err)
	fmt.Println("(err == nil): ", err == nil)

}
