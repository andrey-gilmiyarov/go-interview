package backend

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrContainsDigit     = errors.New("line contains a digit")
	ErrLineTooLong       = errors.New("line has at least 20 Unicode code points")
	ErrRequiresTwoSpaces = errors.New("line must contain exactly two spaces")
)

func ValidateLine(line string) error {
	var problems []error
	for _, r := range line {
		if unicode.IsDigit(r) {
			problems = append(problems, ErrContainsDigit)
			break
		}
	}
	if utf8.RuneCountInString(line) >= 20 {
		problems = append(problems, ErrLineTooLong)
	}
	if strings.Count(line, " ") != 2 {
		problems = append(problems, ErrRequiresTwoSpaces)
	}
	return errors.Join(problems...)
}
