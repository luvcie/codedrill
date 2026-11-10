package reference

import "slices"

// Normalize removes parentheses that only wrap a complete return expression,
// so "return (0);" and "return 0;" produce the same tokens. Parentheses that
// take part in a larger expression, such as "(a + b) * c", are kept.
func Normalize(toks []Token) []Token {
	out := slices.Clone(toks)
	for {
		next, changed := stripReturnParens(out)
		if !changed {
			return out
		}
		out = next
	}
}

// stripReturnParens drops one layer of parentheses around a return
// expression, meaning the matching ")" is followed by ";" and the parentheses
// are not the expression itself.
func stripReturnParens(toks []Token) ([]Token, bool) {
	out := make([]Token, 0, len(toks))
	changed := false
	for i := 0; i < len(toks); i++ {
		if isReturn(toks[i]) && i+1 < len(toks) && isPunct(toks[i+1], "(") {
			if close, ok := matchingParen(toks, i+1); ok && close > i+2 && close+1 < len(toks) && isPunct(toks[close+1], ";") {
				out = append(out, toks[i])
				out = append(out, toks[i+2:close]...)
				i = close
				changed = true
				continue
			}
		}
		out = append(out, toks[i])
	}
	if !changed {
		return toks, false
	}
	return out, true
}

func matchingParen(toks []Token, open int) (int, bool) {
	depth := 0
	for i := open; i < len(toks); i++ {
		if toks[i].Kind != Punct {
			continue
		}
		switch toks[i].Text {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

func isReturn(tok Token) bool { return tok.Kind == Ident && tok.Text == "return" }

func isPunct(tok Token, text string) bool { return tok.Kind == Punct && tok.Text == text }
