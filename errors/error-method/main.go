package main

import "fmt"

type User struct {
	name string
	age  int
}

func (u *User) Error() string {
	return fmt.Sprintf("User %s is %d years old", u.name, u.age)
}

func main() {
	// user := User{name: "Alice", age: 30}
	user := &User{name: "Alice", age: 30}
	fmt.Println("User", user)
}