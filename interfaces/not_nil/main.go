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

func main() {
	// Example 1
	var pointer *MyError = nil
	var err error = pointer
	fmt.Println("nil:", err == nil)

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
