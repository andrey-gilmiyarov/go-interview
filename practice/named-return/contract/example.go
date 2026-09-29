package contract

func example() (result int) {
	defer func() { result++ }()
	return 4
}
