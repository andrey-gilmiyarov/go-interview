package contract

func Observe() Observation {
	nilEqual, emptyEqual, nilJSON, emptyJSON := example()
	return Observation{Answer: Answer{
		NilEqual: nilEqual, EmptyEqual: emptyEqual,
		NilJSON: nilJSON, EmptyJSON: emptyJSON,
	}}
}
