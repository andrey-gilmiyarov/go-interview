package contract

type MyError struct{}

func (*MyError) Error() string { return "ошибка" }

func example() (errorIsNil, pointerIsNil bool) {
	var pointer *MyError
	var err error = pointer
	return err == nil, pointer == nil
}
