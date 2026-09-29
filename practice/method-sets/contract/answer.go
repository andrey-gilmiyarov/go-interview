package contract

// #region answer
type Answer struct {
	ValueImplements         bool
	PointerImplements       bool
	EmbeddedValueImplements bool
}

// #endregion answer

type Observation struct {
	Answer
}
