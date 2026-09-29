package contract

func Observe() Observation {
	actual, leftReady, rightReady, nilSelected, closedValue, closedOK := example()
	return Observation{
		Answer: Answer{
			ReadyOutcomes: []string{"left", "right"},
			NilBlocks:     true,
			ClosedValue:   closedValue,
			ClosedOK:      closedOK,
		},
		ActualReadyOutcome: actual,
		LeftWasReady:       leftReady,
		RightWasReady:      rightReady,
		NilCaseWasSelected: nilSelected,
	}
}
