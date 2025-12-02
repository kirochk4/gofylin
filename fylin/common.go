package fylin

import (
	"fmt"
	"strings"
)

const Version = "0.0.0"

const (
	debugPrintTokens = true
	debugPrintAST    = true
)

type varName = string
type varType string

const (
	varLocal    varType = "local"
	varNonLocal varType = "nonlocal"
	varGlobal   varType = "global"
)

var integerBases = map[byte]int{'x': 16, 'o': 8, 'b': 2}

func cover(str string, width int, c string) string {
	left := (width - len(str)/2) - 1
	var right int
	if len(str)%2 == 0 {
		right = left
	} else {
		right = left + 1
	}
	return fmt.Sprintf(
		"%s %s %s",
		multiplyString(c, left),
		str,
		multiplyString(c, right),
	)
}

func multiplyString(str string, m int) string {
	var res strings.Builder
	for range m {
		res.WriteString(str)
	}
	return res.String()
}

func shortString(str string, length int, cutNewLine bool) string {
	rStr := []rune(str)
	if cutNewLine {
		i := runesIndex(rStr, '\r')
		if i < 0 {
			i = runesIndex(rStr, '\n')
		}
		if i >= 0 && (length < 0 || i < length) {
			return string(rStr[:i])
		}
	}
	if length >= 0 && length < len(rStr) {
		return string(rStr[:length])
	}
	return str
}

func runesIndex(runes []rune, target rune) int {
	for i, r := range runes {
		if r == target {
			return i
		}
	}
	return -1
}

func catch[E any](onCatch func(E)) {
	if p := recover(); p != nil {
		if pe, ok := p.(E); ok {
			onCatch(pe)
		} else {
			panic(p)
		}
	}
}

func lowerChar(char byte) byte { return ('a' - 'A') | char }
