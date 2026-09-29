package contract

func Observe() Observation {
	valueImplements, pointerImplements, embeddedValueImplements := example()
	return Observation{Answer: Answer{
		ValueImplements:         valueImplements,
		PointerImplements:       pointerImplements,
		EmbeddedValueImplements: embeddedValueImplements,
	}}
}
