package reference

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// cppSource keeps physical locations after line splicing. Raw strings read
// their original spelling because C++ reverts splicing inside their quotes.
type cppSource struct {
	text, original string
	offsets, lines []int
}

func spliceCPP(src string) cppSource {
	s := cppSource{original: src}
	var b strings.Builder
	line := 1
	for i := 0; i < len(src); {
		if src[i] == '\\' {
			end := i + 1
			for end < len(src) && isSpace(src[end]) && src[end] != '\r' {
				end++
			}
			if end < len(src) && src[end] == '\r' {
				end++
			}
			if end < len(src) && src[end] == '\n' {
				i = end + 1
				line++
				continue
			}
		}
		s.offsets = append(s.offsets, i)
		s.lines = append(s.lines, line)
		c := src[i]
		if c == '\r' {
			c = '\n'
			if i+1 < len(src) && src[i+1] == '\n' {
				i++
			}
		}
		b.WriteByte(c)
		if c == '\n' {
			line++
		}
		i++
	}
	s.offsets = append(s.offsets, len(src))
	s.lines = append(s.lines, line)
	s.text = b.String()
	return s
}

// TokenizeCPP retains preprocessing token boundaries, literal spellings and
// directive whitespace. It does not parse templates or expand macros. In
// particular, >> stays distinct from > >, and return parentheses are retained.
// Unsupported spellings return an error rather than an uncertain exact match.
func TokenizeCPP(src string) ([]Token, error) {
	if !utf8.ValidString(src) {
		return nil, fmt.Errorf("invalid UTF-8 source")
	}
	s := spliceCPP(src)
	var tokens []Token
	lineStart := true
	exported := false
	for i := 0; i < len(s.text); {
		next, newline, err := s.trivia(i, false)
		if err != nil {
			return nil, err
		}
		lineStart = lineStart || newline
		i = next
		if i == len(s.text) {
			break
		}
		var tok Token
		if s.text[i] == '#' || strings.HasPrefix(s.text[i:], "%:") && !strings.HasPrefix(s.text[i:], "%:%:") {
			if !lineStart {
				return nil, fmt.Errorf("line %d: directive marker inside a source line", s.lines[i])
			}
			tok, next, err = s.directive(i)
		} else {
			tok, next, err = s.token(i)
		}
		if err != nil {
			return nil, err
		}
		if tok.Kind == Ident && (tok.Text == "import" || tok.Text == "module") && (lineStart || exported) {
			return nil, fmt.Errorf("line %d: module directives are unsupported for comparison", tok.Line)
		}
		exported = lineStart && tok.Kind == Ident && tok.Text == "export"
		tokens = append(tokens, tok)
		i = next
		lineStart = false
	}
	return tokens, nil
}

func (s cppSource) trivia(i int, stopAtNewline bool) (int, bool, error) {
	newline := false
	for i < len(s.text) {
		switch {
		case s.text[i] == '\n':
			if stopAtNewline {
				return i, newline, nil
			}
			newline = true
			i++
		case isSpace(s.text[i]):
			i++
		case strings.HasPrefix(s.text[i:], "//"):
			for i < len(s.text) && s.text[i] != '\n' {
				i++
			}
		case strings.HasPrefix(s.text[i:], "/*"):
			end := strings.Index(s.text[i+2:], "*/")
			if end < 0 {
				return 0, false, fmt.Errorf("line %d: unterminated block comment", s.lines[i])
			}
			i += end + 4
		default:
			return i, newline, nil
		}
	}
	return i, newline, nil
}

func (s cppSource) token(i int) (Token, int, error) {
	tok := Token{Line: s.lines[i]}
	text := s.text
	end := i
	quote := i
	for _, prefix := range []string{"u8R", "uR", "UR", "LR", "R", "u8", "u", "U", "L", ""} {
		q := i + len(prefix)
		if strings.HasPrefix(text[i:], prefix) && q < len(text) && (text[q] == '"' || text[q] == '\'') {
			quote = q
			break
		}
	}
	switch {
	case text[quote] == '"' || text[quote] == '\'':
		tok.Kind = String
		if text[quote] == '\'' {
			tok.Kind = Char
		}
		if quote > i && text[quote-1] == 'R' {
			if tok.Kind != String {
				return Token{}, 0, fmt.Errorf("line %d: invalid raw string literal", tok.Line)
			}
			var err error
			end, tok.Text, err = s.raw(i, quote)
			if err != nil {
				return Token{}, 0, err
			}
		} else {
			var err error
			end, err = scanCPPQuoted(text, quote, tok.Line)
			if err != nil {
				return Token{}, 0, err
			}
			if tok.Kind == Char && end == quote+2 {
				return Token{}, 0, fmt.Errorf("line %d: empty character literal", tok.Line)
			}
			tok.Text = text[i:end]
		}
		// A user-defined suffix is part of the literal only when adjacent.
		suffix := end
		if end < len(text) && isIdentStart(text[end]) {
			end++
			for end < len(text) && isIdentPart(text[end]) {
				end++
			}
		}
		tok.Text += text[suffix:end]
	case isDigit(text[i]) || text[i] == '.' && i+1 < len(text) && isDigit(text[i+1]):
		tok.Kind = Number
		end = i + 1
		for end < len(text) {
			c := text[end]
			if isIdentPart(c) || c == '.' || (c == '+' || c == '-') && isExponentMark(text[end-1]) {
				end++
			} else if c == '\'' && end+1 < len(text) && isIdentPart(text[end+1]) {
				end += 2
			} else {
				break
			}
		}
	case isIdentStart(text[i]):
		tok.Kind = Ident
		end = i + 1
		for end < len(text) && isIdentPart(text[end]) {
			end++
		}
		if text[i:end] == "__LINE__" {
			// Moving this token can change the program despite identical spelling.
			tok.Text = fmt.Sprintf("__LINE__@%d", tok.Line)
		}
	default:
		tok.Kind = Punct
		punct := ""
		// C++'s <:: exception avoids consuming <: in A<::B>.
		if strings.HasPrefix(text[i:], "<::") && (i+3 == len(text) || text[i+3] != ':' && text[i+3] != '>') {
			punct = "<"
		} else {
			for _, p := range cppPunctuators {
				if strings.HasPrefix(text[i:], p) {
					punct = p
					break
				}
			}
		}
		if punct == "" {
			return Token{}, 0, fmt.Errorf("line %d: unsupported source character %q", tok.Line, text[i])
		}
		end = i + len(punct)
	}
	if tok.Text == "" {
		tok.Text = text[i:end]
	}
	return tok, end, nil
}

// Line splices have already been removed; a remaining newline cannot occur
// inside an ordinary literal. Unknown escape forms fail closed.
func scanCPPQuoted(src string, start, line int) (int, error) {
	quote := src[start]
	for i := start + 1; i < len(src); i++ {
		switch src[i] {
		case quote:
			return i + 1, nil
		case '\n':
			return 0, unterminatedLiteral(quote, line)
		case '\\':
			i++
			if i == len(src) {
				return 0, unterminatedLiteral(quote, line)
			}
			switch c := src[i]; {
			case strings.ContainsRune("'\"?\\abfnrtv", rune(c)):
			case c >= '0' && c <= '7':
				for n := 1; n < 3 && i+1 < len(src) && src[i+1] >= '0' && src[i+1] <= '7'; n++ {
					i++
				}
			case c == 'x':
				if i+1 == len(src) || !isHexDigit(src[i+1]) {
					return 0, fmt.Errorf("line %d: incomplete hexadecimal escape", line)
				}
				for i+1 < len(src) && isHexDigit(src[i+1]) {
					i++
				}
			case c == 'u' || c == 'U':
				digits := 4
				if c == 'U' {
					digits = 8
				}
				for n := 0; n < digits; n++ {
					i++
					if i == len(src) || !isHexDigit(src[i]) {
						return 0, fmt.Errorf("line %d: unsupported or incomplete universal character escape", line)
					}
				}
			default:
				return 0, fmt.Errorf("line %d: unsupported escape sequence", line)
			}
		}
	}
	return 0, unterminatedLiteral(quote, line)
}

func isHexDigit(c byte) bool {
	return isDigit(c) || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func (s cppSource) raw(start, quote int) (int, string, error) {
	q := s.offsets[quote]
	open := q + 1
	for open < len(s.original) && s.original[open] != '(' {
		c := s.original[open]
		if open-q > 16 || c <= ' ' || c > '~' || c == ')' || c == '\\' {
			return 0, "", fmt.Errorf("line %d: invalid raw string delimiter", s.lines[start])
		}
		open++
	}
	if open == len(s.original) || open-q-1 > 16 {
		return 0, "", fmt.Errorf("line %d: invalid raw string delimiter", s.lines[start])
	}
	close := ")" + s.original[q+1:open] + "\""
	n := strings.Index(s.original[open+1:], close)
	if n < 0 {
		return 0, "", fmt.Errorf("line %d: unterminated raw string literal", s.lines[start])
	}
	end := open + 1 + n + len(close)
	return sort.SearchInts(s.offsets, end), s.text[start:quote] + s.original[q:end], nil
}

func (s cppSource) directive(start int) (Token, int, error) {
	marker, i, err := s.token(start)
	if err != nil {
		return Token{}, 0, err
	}
	var b strings.Builder
	b.WriteString(marker.Text)
	first := true
	headerPending := false
	for i < len(s.text) {
		next, _, err := s.trivia(i, true)
		if err != nil {
			return Token{}, 0, err
		}
		if next == len(s.text) || s.text[next] == '\n' {
			return Token{Kind: Directive, Text: b.String(), Line: s.lines[start]}, next, nil
		}
		space := next != i
		i = next
		var tok Token
		if headerPending && s.text[i] == '<' {
			end := strings.IndexAny(s.text[i+1:], ">\n")
			if end < 0 || s.text[i+1+end] != '>' {
				return Token{}, 0, fmt.Errorf("line %d: unterminated header name", s.lines[i])
			}
			next = i + end + 2
			tok.Text = s.text[i:next]
		} else {
			tok, next, err = s.token(i)
			if err != nil {
				return Token{}, 0, err
			}
			// Stringification makes whitespace in macro arguments observable.
			// Without macro expansion it cannot safely be discarded.
			if tok.Text == "#" || tok.Text == "%:" {
				return Token{}, 0, fmt.Errorf("line %d: stringifying macros are unsupported for comparison", tok.Line)
			}
		}
		if space && !first {
			b.WriteByte(' ')
		}
		b.WriteString(tok.Text)
		if first && (tok.Text == "include" || tok.Text == "include_next" || tok.Text == "embed") || tok.Text == "__has_include" || tok.Text == "__has_embed" {
			headerPending = true
		} else if tok.Text != "(" {
			headerPending = false
		}
		first = false
		i = next
	}
	return Token{Kind: Directive, Text: b.String(), Line: s.lines[start]}, i, nil
}

var cppPunctuators = []string{
	"%:%:", "<=>", "->*", "<<=", ">>=", "...",
	"::", ".*", "<:", ":>", "<%", "%>", "%:",
	"->", "++", "--", "<<", ">>", "<=", ">=", "==", "!=", "&&", "||",
	"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^=", "##",
	"+", "-", "*", "/", "%", "<", ">", "=", "!", "&", "|", "^", "~",
	"?", ":", ";", ",", ".", "(", ")", "[", "]", "{", "}", "#",
}
