package contract

// #region answer
type Answer struct {
	NilEqual   bool
	EmptyEqual bool
	NilJSON    string
	EmptyJSON  string
}

// #endregion answer

type Observation struct {
	Answer
}
