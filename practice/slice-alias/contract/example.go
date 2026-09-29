package contract

func example() (shared, detached []int) {
	a := []int{1, 2, 3}
	s := a[:2]
	s = append(s, 9)
	shared = cloneInts(a)

	s = append(s, 10)
	s[0] = 7
	detached = cloneInts(a)
	return shared, detached
}

func cloneInts(values []int) []int {
	return append([]int(nil), values...)
}
