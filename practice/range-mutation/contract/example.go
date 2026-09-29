package contract

func example() (seen, final []int) {
	x := []int{1, 2, 3}
	for _, value := range x {
		seen = append(seen, value)
		x = append(x, value)
	}
	return seen, x
}
