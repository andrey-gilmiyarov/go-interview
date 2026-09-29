package contract

import "encoding/json"

func example() (nilEqual, emptyEqual bool, nilJSON, emptyJSON string) {
	var nilSlice []int
	emptySlice := make([]int, 0)

	nilEqual = nilSlice == nil
	emptyEqual = emptySlice == nil
	nilBytes, _ := json.Marshal(nilSlice)
	emptyBytes, _ := json.Marshal(emptySlice)
	return nilEqual, emptyEqual, string(nilBytes), string(emptyBytes)
}
