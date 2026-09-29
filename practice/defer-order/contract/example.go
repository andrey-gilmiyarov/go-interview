package contract

import "strconv"

func example() (events []string) {
	x := 1
	log := func(value int) { events = append(events, strconv.Itoa(value)) }

	defer log(x)
	defer func() { log(x) }()
	x = 2
	return
}
