package contract

func Observe() Observation {
	shared, detached := example()
	return Observation{Answer: Answer{Shared: shared, Detached: detached}}
}
