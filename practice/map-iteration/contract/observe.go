package contract

func Observe() Observation {
	actual := example()
	return Observation{
		Answer: Answer{
			OrderGuaranteed:               false,
			MustVisitInserted:             false,
			MustVisitDeletedBeforeReached: false,
		},
		InitialKeys:           actual.initialKeys,
		InitialVisits:         actual.initialVisits,
		InsertedKeyWasVisited: actual.insertedKeyWasVisited,
		DeletedBeforeReached:  actual.deletedBeforeReached,
		DeletedKeyWasVisited:  actual.deletedKeyWasVisited,
	}
}
