package contract

func Observe() Observation {
	return Observation{Answer: Answer{Result: example()}}
}
