package contract

// #region answer
type Answer struct {
	Declared []int
	Reused   []int
}

// #endregion answer

type Observation struct {
	Answer
}
