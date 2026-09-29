package contract

// #region answer
type Answer struct {
	ReadyOutcomes []string
	NilBlocks     bool
	ClosedValue   int
	ClosedOK      bool
}

// #endregion answer

type Observation struct {
	Answer
	ActualReadyOutcome string
	LeftWasReady       bool
	RightWasReady      bool
	NilCaseWasSelected bool
}
