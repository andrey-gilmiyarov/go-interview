package contract

func Observe() Observation {
	declared, reused := example()
	return Observation{Answer: Answer{Declared: declared, Reused: reused}}
}
