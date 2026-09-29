package contract

func Observe() Observation {
	seen, final := example()
	return Observation{Answer: Answer{Seen: seen, Final: final}}
}
