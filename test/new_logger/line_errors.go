package new_logger

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"
)

var ErrFoundNum = errors.New("found numbers")
var ErrLongLine = errors.New("line is too long")
var ErrNoTwoSpaces = errors.New("no two spaces")

type LineErrors []error

func (le LineErrors) Error() string {
	var sb strings.Builder
	for _, l := range le {
		sb.WriteString(fmt.Sprintf("%v\n", l.Error()))
	}
	return sb.String()
}

func MyCheck(ret string) error {
	lineErrors := LineErrors{}
	spaces := 0
	for _, r := range ret {
		if !unicode.IsLetter(r) {
			lineErrors = append(lineErrors, ErrFoundNum)
		}
		if r == ' ' {
			spaces++
		}
	}
	if len([]rune(ret)) >= 20 {
		lineErrors = append(lineErrors, ErrLongLine)
	}
	if spaces != 2 {
		lineErrors = append(lineErrors, ErrNoTwoSpaces)
	}
	return lineErrors
}

func StartCheck() {
	for {
		fmt.Printf("Укажите строку (q для выхода): ")
		reader := bufio.NewReader(os.Stdin)
		ret, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			continue
		}
		ret = strings.TrimRight(ret, "\n")
		if ret == `q` {
			break
		}
		if err = MyCheck(ret); err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(`Строка прошла проверку`)
		}
	}
}
