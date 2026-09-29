package contract

func example() (declared, reused []int) {
	var declaredClosures []func() int
	for i := 0; i < 3; i++ {
		declaredClosures = append(declaredClosures, func() int { return i })
	}
	for _, closure := range declaredClosures {
		declared = append(declared, closure())
	}

	var i int
	var reusedClosures []func() int
	for i = 0; i < 3; i++ {
		reusedClosures = append(reusedClosures, func() int { return i })
	}
	for _, closure := range reusedClosures {
		reused = append(reused, closure())
	}
	return declared, reused
}
