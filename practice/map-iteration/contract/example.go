package contract

type mapObservation struct {
	initialKeys           []string
	initialVisits         map[string]int
	insertedKeyWasVisited bool
	deletedBeforeReached  bool
	deletedKeyWasVisited  bool
}

func example() mapObservation {
	initial := map[string]int{"alpha": 1, "beta": 2, "gamma": 3}
	visits := make(map[string]int, len(initial))
	for key := range initial {
		visits[key]++
	}

	inserted := map[string]bool{"trigger": true}
	insertedSeen := false
	for key := range inserted {
		if key == "trigger" {
			inserted["new"] = true
		}
		if key == "new" {
			insertedSeen = true
		}
	}

	deleted := map[string]bool{"target": true, "trigger": true}
	targetSeen := false
	deletedBeforeReach := false
	deletedSeen := false
	for key := range deleted {
		if key == "target" {
			targetSeen = true
			if deletedBeforeReach {
				deletedSeen = true
			}
		}
		if key == "trigger" && !targetSeen {
			delete(deleted, "target")
			deletedBeforeReach = true
		}
	}

	keys := make([]string, 0, len(initial))
	for key := range initial {
		keys = append(keys, key)
	}
	return mapObservation{
		initialKeys: keys, initialVisits: visits,
		insertedKeyWasVisited: insertedSeen,
		deletedBeforeReached:  deletedBeforeReach,
		deletedKeyWasVisited:  deletedSeen,
	}
}
