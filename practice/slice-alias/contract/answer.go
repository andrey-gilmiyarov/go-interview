package contract

// #region answer
type Answer struct {
	Shared   []int
	Detached []int
}

// #endregion answer

type Observation struct {
	Answer
}
