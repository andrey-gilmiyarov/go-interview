package contract

// #region answer
type Answer struct {
	OrderGuaranteed               bool
	MustVisitInserted             bool
	MustVisitDeletedBeforeReached bool
}

// #endregion answer

type Observation struct {
	Answer
	InitialKeys           []string
	InitialVisits         map[string]int
	InsertedKeyWasVisited bool
	DeletedBeforeReached  bool
	DeletedKeyWasVisited  bool
}
