package fylin

import (
	"fmt"
	"strings"
)

const eofByte byte = 0

type tokenType string

const (
	// single
	tokenLeftParen    tokenType = "left paren"
	tokenRightParen   tokenType = "right paren"
	tokenLeftBrace    tokenType = "left brace"
	tokenRightBrace   tokenType = "right brace"
	tokenLeftBracket  tokenType = "left bracket"
	tokenRightBracket tokenType = "right bracket"
	tokenComma        tokenType = "comma"
	tokenMinus        tokenType = "minus"
	tokenPlus         tokenType = "plus"
	tokenSlash        tokenType = "slash"
	tokenStar         tokenType = "star"
	tokenDot          tokenType = "dot"
	tokenColon        tokenType = "colon"
	tokenDog          tokenType = "dog"
	// double
	tokenBangEqual    tokenType = "bang equal"
	tokenEqual        tokenType = "equal"
	tokenEqualEqual   tokenType = "equal equal"
	tokenGreater      tokenType = "greater"
	tokenGreaterEqual tokenType = "greater equal"
	tokenLess         tokenType = "less"
	tokenLessEqual    tokenType = "less equal"
	// special
	tokenIdentifier tokenType = "identifier"
	tokenString     tokenType = "string"
	tokenFloat      tokenType = "float"
	tokenInteger    tokenType = "integer"
	// keywords
	tokenBreak    tokenType = "break"
	tokenContinue tokenType = "continue"
	tokenAnd      tokenType = "and"
	tokenElse     tokenType = "else"
	tokenFalse    tokenType = "false"
	tokenDef      tokenType = "def"
	tokenIf       tokenType = "if"
	tokenElif     tokenType = "elif"
	tokenNone     tokenType = "none"
	tokenOr       tokenType = "or"
	tokenReturn   tokenType = "return"
	tokenTrue     tokenType = "true"
	tokenWhile    tokenType = "while"
	tokenFor      tokenType = "for"
	tokenNot      tokenType = "not"
	tokenPass     tokenType = "pass"
	tokenNonLocal tokenType = "non local"
	tokenLocal    tokenType = "local"
	tokenGlobal   tokenType = "global"
	tokenIn       tokenType = "in"
	tokenRaise    tokenType = "raise"
	tokenTry      tokenType = "try"
	tokenExcept   tokenType = "except"
	tokenFinally  tokenType = "finally"
	tokenAs       tokenType = "as"

	tokenNewLine tokenType = "new line"
	tokenIntab   tokenType = "intab"
	tokenDetab   tokenType = "detab"

	tokenError tokenType = "error"
	tokenEof   tokenType = "eof"
)

type token struct {
	tokenType
	line    int
	literal string
}

func (t token) String() string {
	return fmt.Sprintf(
		"%04d %-12s '%s'",
		t.line,
		t.tokenType,
		shortString(t.literal, 32, true),
	)
}

type tabType = int

const (
	tabSpace = 0b01
	tabTab   = 0b10

	tabMixed = tabSpace | tabTab
	tabNone  = tabSpace & tabTab
)

type scanner struct {
	source  []byte
	sp      int // source pointer
	start   int
	line    int
	newLine bool
	tabs    []int
	curTab  int
	tabType
}

func newScanner(source []byte) scanner {
	return scanner{
		source:  source,
		sp:      0,
		line:    1,
		newLine: false,
		curTab:  0,
		tabs:    []int{0},
	}
}

func (s *scanner) scanToken() token {
	startLine := s.line
	s.skipWhitespace()

	s.start = s.sp

	if s.newLine && (startLine < s.line || s.isAtEnd()) {
		return s.makeToken(tokenNewLine)
	}

	if s.curTab < s.lastTab() {
		s.detab()
		if s.curTab > s.lastTab() {
			return s.errorToken("wrong detab size")
		}
		if s.curTab == 0 {
			s.tabType = tabNone
		}
		return s.makeToken(tokenDetab)
	} else if s.curTab > s.lastTab() {
		if s.curTab == tabMixed {
			s.curTab = tabNone
			return s.errorToken("mixed tabs and spaces")
		}
		s.intab()
		return s.makeToken(tokenIntab)
	}

	if s.isAtEnd() {
		if len(s.tabs) != 1 {
			s.detab()
			s.curTab = 0
			return s.makeToken(tokenDetab)
		}
		return s.makeToken(tokenEof)
	}

	char := s.advance()

	if isAlpha(char) {
		return s.identifier()
	}
	if isDigit(char) {
		if char == '0' {
			switch lowerChar(s.current()) {
			case 'x':
				return s.integer(16)
			case 'o':
				return s.integer(8)
			case 'b':
				return s.integer(2)
			}
		}
		return s.float()
	}
	switch char {
	case '(':
		return s.makeToken(tokenLeftParen)
	case ')':
		return s.makeToken(tokenRightParen)
	case '{':
		return s.makeToken(tokenLeftBrace)
	case '}':
		return s.makeToken(tokenRightBrace)
	case '[':
		return s.makeToken(tokenLeftBracket)
	case ']':
		return s.makeToken(tokenRightBracket)
	case ',':
		return s.makeToken(tokenComma)
	case '-':
		return s.makeToken(tokenMinus)
	case '+':
		return s.makeToken(tokenPlus)
	case '/':
		return s.makeToken(tokenSlash)
	case '*':
		return s.makeToken(tokenStar)
	case ':':
		return s.makeToken(tokenColon)
	case '.':
		return s.makeToken(tokenDot)
	case '@':
		return s.makeToken(tokenDog)
	case '!':
		if s.match('=') {
			return s.makeToken(tokenBangEqual)
		}
	case '=':
		t := tokenEqual
		if s.match('=') {
			t = tokenEqualEqual
		}
		return s.makeToken(t)
	case '<':
		t := tokenLess
		if s.match('=') {
			t = tokenLessEqual
		}
		return s.makeToken(t)
	case '>':
		t := tokenGreater
		if s.match('=') {
			t = tokenGreaterEqual
		}
		return s.makeToken(t)
	case '"':
		return s.string()
	}
	return s.errorToken("Unexpected character.")
}

func (s *scanner) intab()       { s.tabs = append(s.tabs, s.curTab) }
func (s *scanner) detab()       { s.tabs = s.tabs[:len(s.tabs)-1] }
func (s *scanner) lastTab() int { return s.tabs[len(s.tabs)-1] }

func (s *scanner) isAtEnd() bool {
	return s.current() == eofByte
}

func (s *scanner) current() byte {
	if s.sp >= len(s.source) {
		return eofByte
	}
	return s.source[s.sp]
}

func (s *scanner) previous() byte {
	if s.sp-1 < 0 {
		return eofByte
	}
	return s.source[s.sp-1]
}

func (s *scanner) peek() byte {
	if s.sp+1 >= len(s.source) {
		return eofByte
	}
	return s.source[s.sp+1]
}

func (s *scanner) advance() byte {
	ret := s.current()
	s.sp++
	return ret
}

func (s *scanner) match(expect byte) bool {
	if s.isAtEnd() {
		return false
	}
	if s.current() != expect {
		return false
	}
	s.sp++
	return true
}

func (s *scanner) makeToken(t tokenType) token {
	switch t {
	case tokenIdentifier, tokenFloat, tokenInteger, tokenString,
		tokenNone, tokenFalse, tokenTrue,
		tokenBreak, tokenContinue, tokenReturn,
		tokenRightParen, tokenRightBracket, tokenRightBrace,
		tokenColon:
		s.newLine = true
	default:
		s.newLine = false
	}
	var literal string
	if t == tokenString {
		if l, ok := escapeString(s.source[s.start:s.sp]); !ok {
			return s.errorToken("Invalid escape sequence.")
		} else {
			literal = l
		}
	} else {
		literal = string(s.source[s.start:s.sp])
	}
	tk := token{t, s.line, literal}
	if debugPrintTokens {
		fmt.Println(tk)
	}
	return tk
}

func (s *scanner) errorToken(message string) token {
	return token{
		tokenType: tokenError,
		line:      s.line,
		literal:   message,
	}
}

func (s *scanner) skipWhitespace() {
	line := s.line
	var tab int
	defer func() {
		if s.line > line {
			s.curTab = tab
		}
	}()
	for {
		char := s.current()
		switch char {
		case ' ':
			s.tabType |= tabSpace
			tab++
			s.advance()
		case '\t':
			s.tabType |= tabTab
			tab++
			s.advance()
		case '\r':
			s.advance()
		case '\n':
			tab = 0
			s.line++
			s.advance()
		case '/':
			if s.peek() == '/' {
				for s.current() != '\n' && !s.isAtEnd() {
					s.advance()
				}
			} else {
				return
			}
		default:
			return
		}
	}
}

func (s *scanner) identifierType() tokenType {
	if t, ok := keywords[string(s.source[s.start:s.sp])]; ok {
		return t
	}
	return tokenIdentifier
}

func (s *scanner) identifier() token {
	for isAlpha(s.current()) || isDigit(s.current()) {
		s.advance()
	}
	return s.makeToken(s.identifierType())
}

func (s *scanner) float() token {
	allowUnderscore := true
	for isDigit(s.current()) || (allowUnderscore && s.current() == '_') {
		allowUnderscore = s.current() != '_'
		s.advance()
	}
	if s.current() == '.' && isDigit(s.peek()) && allowUnderscore {
		s.advance()
		for isDigit(s.current()) || (allowUnderscore && s.current() == '_') {
			allowUnderscore = s.current() != '_'
			s.advance()
		}
	}
	if !allowUnderscore {
		if s.current() == '_' {
			return s.errorToken("Double underscore in number literal.")
		}
		return s.errorToken("Number literal ends with underscore.")
	}
	return s.makeToken(tokenFloat)
}

func (s *scanner) integer(base int) token {
	s.advance()
	if base <= 10 {
		max := byte('0' + base)
		for isDigit(s.current()) {
			if s.current() >= max {
				return s.errorToken("Invalid symbol in number literal.")
			}
			s.advance()
		}
	} else {
		for isHex(s.current()) {
			s.advance()
		}
	}
	return s.makeToken(tokenInteger)
}

func (s *scanner) string() token {
	for !(s.current() == '"' && s.previous() != '\\') && !s.isAtEnd() {
		if s.current() == '\n' {
			s.line++
		}
		s.advance()
	}
	if s.isAtEnd() {
		return s.errorToken("Unterminated string.")
	}
	s.advance()
	return s.makeToken(tokenString)
}

func isAlpha(char byte) bool {
	return 'a' <= char && char <= 'z' ||
		'A' <= char && char <= 'Z' ||
		char == '_'
}

func isDigit(char byte) bool {
	return '0' <= char && char <= '9'
}

func isHex(char byte) bool {
	return isDigit(char) || 'a' <= lowerChar(char) && lowerChar(char) <= 'f'
}

func escapeString(source []byte) (string, bool) {
	var result strings.Builder
	for i := 0; i < len(source); i++ {
		b := source[i]
		if b == '\\' {
			if i+1 == len(source) {
				return "", false
			}
			switch source[i+1] {
			case 'n':
				result.WriteByte('\n')
			case 'r':
				result.WriteByte('\r')
			case 't':
				result.WriteByte('\t')
			case 'b':
				result.WriteByte('\b')
			case '\\':
				result.WriteByte('\\')
			case '"':
				result.WriteByte('"')
			default:
				return "", false
			}
			i++
			continue
		}
		result.WriteByte(b)
	}
	return result.String(), true
}

var keywords = map[string]tokenType{
	"and":      tokenAnd,
	"else":     tokenElse,
	"not":      tokenNot,
	"False":    tokenFalse,
	"def":      tokenDef,
	"if":       tokenIf,
	"elif":     tokenElif,
	"None":     tokenNone,
	"or":       tokenOr,
	"return":   tokenReturn,
	"True":     tokenTrue,
	"while":    tokenWhile,
	"for":      tokenFor,
	"break":    tokenBreak,
	"continue": tokenContinue,
	"nonlocal": tokenNonLocal,
	"local":    tokenLocal,
	"global":   tokenGlobal,
	"in":       tokenIn,
	"raise":    tokenRaise,
	"try":      tokenTry,
	"except":   tokenExcept,
	"finally":  tokenFinally,
	"as":       tokenAs,
	"pass":     tokenPass,
}
