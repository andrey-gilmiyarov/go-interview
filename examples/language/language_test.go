package language

import (
	"errors"
	"reflect"
	"testing"
)

func TestSliceParameterExample(t *testing.T) {
	got := SliceParameterExample()
	if !reflect.DeepEqual(got.AfterMutation, []int{1, 2, 5, 4}) || !reflect.DeepEqual(got.AfterAppendCall, []int{1, 2, 5, 4}) || !reflect.DeepEqual(got.Resliced, []int{1, 2, 5, 4, 6}) {
		t.Fatalf("SliceParameterExample() = %#v", got)
	}
}

func TestAppendHeaderExamples(t *testing.T) {
	got := BadAppendScenarios()
	want := []AppendScenario{{AfterCall: []int{1}, Resliced: []int{1, 2, 3}}, {AfterCall: []int{1}, Resliced: []int{1}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BadAppendScenarios() = %#v, want %#v", got, want)
	}
}

func TestRepeatedOverlappingAppend(t *testing.T) {
	got := RepeatedOverlappingAppend()
	if got.Original != [4]int{0, 2, 3, 3} || !reflect.DeepEqual(got.Result, []int{0, 2, 3, 3, 3}) {
		t.Fatalf("RepeatedOverlappingAppend() = %#v", got)
	}
}

func TestComplicatedRangeMutation(t *testing.T) {
	got := ComplicatedRangeMutation()
	wantSeen := []RangeVisit{{Index: 0, Value: "A"}, {Index: 1, Value: "M"}, {Index: 2, Value: "C"}}
	wantFinal := []string{"A", "Z", "Z", "Z", "Z", "Z"}
	if !reflect.DeepEqual(got.Seen, wantSeen) || !reflect.DeepEqual(got.Final, wantFinal) {
		t.Fatalf("ComplicatedRangeMutation() = %#v", got)
	}
}

func TestDeferExamples(t *testing.T) {
	if got := DeferArgumentTrace(); !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Fatalf("DeferArgumentTrace() = %v", got)
	}
	if got := DeferReceiverTrace(); !reflect.DeepEqual(got, []int{200, 0}) {
		t.Fatalf("DeferReceiverTrace() = %v", got)
	}
	first, nested := DeferredOrderTraces()
	if !reflect.DeepEqual(first, []int{1, 2, 3}) || !reflect.DeepEqual(nested, []int{1, 2, 3, 4}) {
		t.Fatalf("DeferredOrderTraces() = %v, %v", first, nested)
	}
}

func TestUserErrorStory(t *testing.T) {
	err := UserErrorExample()
	if err == nil || err.Error() != "User Alice is 30 years old" {
		t.Fatalf("UserErrorExample() = %v", err)
	}
	var user *UserError
	if !errors.As(err, &user) || user.name != "Alice" || user.age != 30 {
		t.Fatalf("UserErrorExample() has wrong concrete error: %#v", user)
	}
}

func TestSelectDrainExample(t *testing.T) {
	got := SelectDrainExample()
	want := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectDrainExample() = %v, want %v", got, want)
	}
}

func TestTypedNilUserError(t *testing.T) {
	err := TypedNilUserError()
	if err == nil {
		t.Fatal("typed nil *UserError was converted to a nil error interface")
	}
	var user *UserError
	if !errors.As(err, &user) || user != nil {
		t.Fatalf("typed nil conversion = %#v", user)
	}
}
