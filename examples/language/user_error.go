package language

import "fmt"

type UserError struct {
	name string
	age  int
}

func (user *UserError) Error() string {
	return fmt.Sprintf("User %s is %d years old", user.name, user.age)
}

func UserErrorExample() error {
	user := &UserError{name: "Alice", age: 30}
	return user
}

func TypedNilUserError() error {
	var user *UserError
	return user
}
