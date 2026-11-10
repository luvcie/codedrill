// Package reference compares C and C++ source code with a reference solution. The
// comparison works on tokens, so layout and comments do not matter.
package reference

import (
	"fmt"
	"strings"
)

// Kind is the lexical category of a token.
type Kind int

const (
	Ident Kind = iota
	Number
	String
	Char
	Punct
	Directive
)

// Token is one lexical unit of source code. Line is the line the token
// starts on in the source it came from.
type Token struct {
	Kind Kind
	Text string
	Line int
}

// Tokenize splits C source into tokens, dropping whitespace and comments.
// Preprocessor directives, continuations included, are single Directive
// tokens. Malformed input returns an error naming its line.
func Tokenize(src string) ([]Token, error) {
	var toks []Token
	line := 1
	lineStart := true
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '\n':
			line++
			i++
			lineStart = true
		case isSpace(c):
			i++
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			start := line
			i += 2
			closed := false
			for i < len(src) {
				if src[i] == '*' && i+1 < len(src) && src[i+1] == '/' {
					i += 2
					closed = true
					break
				}
				if src[i] == '\n' {
					line++
					lineStart = true
				}
				i++
			}
			if !closed {
				return nil, fmt.Errorf("line %d: unterminated block comment", start)
			}
		case c == '#' && lineStart:
			tok, next, newlines, err := scanDirective(src, i, line)
			if err != nil {
				return nil, err
			}
			toks = append(toks, tok)
			i = next
			line += newlines
			lineStart = true
		case c == '"' || c == '\'':
			start := line
			end, newlines, err := scanQuoted(src, i, line)
			if err != nil {
				return nil, err
			}
			kind := String
			if c == '\'' {
				kind = Char
			}
			toks = append(toks, Token{Kind: kind, Text: src[i:end], Line: start})
			i = end
			line += newlines
			lineStart = false
		case isDigit(c) || (c == '.' && i+1 < len(src) && isDigit(src[i+1])):
			end := scanNumber(src, i)
			toks = append(toks, Token{Kind: Number, Text: src[i:end], Line: line})
			i = end
			lineStart = false
		case isIdentStart(c):
			start := line
			end := i + 1
			for end < len(src) && isIdentPart(src[end]) {
				end++
			}
			kind := Ident
			text := src[i:end]
			if end < len(src) && (src[end] == '"' || src[end] == '\'') && isLiteralPrefix(text) {
				litEnd, newlines, err := scanQuoted(src, end, line)
				if err != nil {
					return nil, err
				}
				kind = String
				if src[end] == '\'' {
					kind = Char
				}
				text = src[i:litEnd]
				end = litEnd
				line += newlines
			}
			toks = append(toks, Token{Kind: kind, Text: text, Line: start})
			i = end
			lineStart = false
		default:
			punct, ok := scanPunct(src[i:])
			if !ok {
				return nil, fmt.Errorf("line %d: unexpected character %q", line, c)
			}
			toks = append(toks, Token{Kind: Punct, Text: punct, Line: line})
			i += len(punct)
			lineStart = false
		}
	}
	return toks, nil
}

// scanDirective reads a preprocessor directive as one token. Directives end at
// an unescaped newline, so a continuation adds to the same token. Runs of
// whitespace outside literals become one space, which keeps the difference
// between "#define F(x)" and "#define F (x)".
func scanDirective(src string, start, line int) (Token, int, int, error) {
	var b strings.Builder
	b.WriteByte('#')
	i := start + 1
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	newlines := 0
	space := false
	write := func(s string) {
		if space && b.Len() > 1 {
			b.WriteByte(' ')
		}
		space = false
		b.WriteString(s)
	}
	for i < len(src) {
		c := src[i]
		switch {
		case c == '\n':
			return Token{Kind: Directive, Text: b.String(), Line: line}, i, newlines, nil
		case c == '\\' && i+1 < len(src) && src[i+1] == '\n':
			i += 2
			newlines++
			space = true
		case isSpace(c):
			i++
			space = true
		case c == '/' && i+1 < len(src) && src[i+1] == '*':
			commentLine := line + newlines
			i += 2
			closed := false
			for i < len(src) {
				if src[i] == '*' && i+1 < len(src) && src[i+1] == '/' {
					i += 2
					closed = true
					break
				}
				if src[i] == '\n' {
					newlines++
				}
				i++
			}
			if !closed {
				return Token{}, 0, 0, fmt.Errorf("line %d: unterminated block comment", commentLine)
			}
			space = true
		case c == '/' && i+1 < len(src) && src[i+1] == '/':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '"' || c == '\'':
			end, extra, err := scanQuoted(src, i, line+newlines)
			if err != nil {
				return Token{}, 0, 0, err
			}
			newlines += extra
			write(src[i:end])
			i = end
		default:
			write(string(c))
			i++
		}
	}
	return Token{Kind: Directive, Text: b.String(), Line: line}, i, newlines, nil
}

// scanQuoted returns the index after a string or character literal and the
// number of newlines skipped by a backslash continuation.
func scanQuoted(src string, start, line int) (int, int, error) {
	quote := src[start]
	i := start + 1
	newlines := 0
	for i < len(src) {
		switch src[i] {
		case '\\':
			if i+1 >= len(src) {
				return 0, 0, unterminatedLiteral(quote, line)
			}
			if src[i+1] == '\n' {
				newlines++
			}
			i += 2
		case quote:
			return i + 1, newlines, nil
		case '\n':
			return 0, 0, unterminatedLiteral(quote, line)
		default:
			i++
		}
	}
	return 0, 0, unterminatedLiteral(quote, line)
}

func unterminatedLiteral(quote byte, line int) error {
	kind := "string"
	if quote == '\'' {
		kind = "character"
	}
	return fmt.Errorf("line %d: unterminated %s literal", line, kind)
}

func scanNumber(src string, start int) int {
	i := start
	for i < len(src) {
		c := src[i]
		switch {
		case isDigit(c) || isLetter(c) || c == '_' || c == '.':
			i++
		case (c == '+' || c == '-') && i > start && isExponentMark(src[i-1]):
			i++
		default:
			return i
		}
	}
	return i
}

// scanPunct matches the longest C operator or punctuation at the start of s.
func scanPunct(s string) (string, bool) {
	for _, p := range punctuators {
		if strings.HasPrefix(s, p) {
			return p, true
		}
	}
	return "", false
}

var punctuators = []string{
	"<<=", ">>=", "...",
	"->", "++", "--", "<<", ">>", "<=", ">=", "==", "!=", "&&", "||",
	"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "##",
	"+", "-", "*", "/", "%", "<", ">", "=", "!", "&", "|", "^", "~",
	"?", ":", ";", ",", ".", "(", ")", "[", "]", "{", "}",
}

func isLiteralPrefix(text string) bool {
	switch text {
	case "L", "u", "U", "u8":
		return true
	}
	return false
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\v' || c == '\f'
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isLetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isExponentMark(c byte) bool {
	return c == 'e' || c == 'E' || c == 'p' || c == 'P'
}

func isIdentStart(c byte) bool { return isLetter(c) || c == '_' }

func isIdentPart(c byte) bool { return isIdentStart(c) || isDigit(c) }
