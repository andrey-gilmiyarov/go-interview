package contract

func Observe() Observation {
	return Observation{Answer: Answer{Events: example()}}
}
