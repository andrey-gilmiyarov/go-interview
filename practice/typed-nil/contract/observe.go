package contract

func Observe() Observation {
	errorIsNil, pointerIsNil := example()
	return Observation{Answer: Answer{
		ErrorIsNil: errorIsNil, PointerIsNil: pointerIsNil,
	}}
}
