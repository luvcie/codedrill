package reference

import (
	"strings"
	"testing"
)

func findToken(toks []Token, text string) (Token, bool) {
	for _, tok := range toks {
		if tok.Text == text {
			return tok, true
		}
	}
	return Token{}, false
}

func TestTokenizeDirectives(t *testing.T) {
	src := "#  include   <stdio.h>\n#define SUM(a, b) \\\n    ((a) + (b))\nint x;\n"
	toks, err := Tokenize(src)
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	want := []string{"#include <stdio.h>", "#define SUM(a, b) ((a) + (b))", "int", "x", ";"}
	if len(toks) != len(want) {
		t.Fatalf("got %d tokens, want %d: %+v", len(toks), len(want), toks)
	}
	for i, text := range want {
		if toks[i].Text != text || toks[i].Kind != Directive && i < 2 {
			t.Fatalf("token %d is %+v, want %q", i, toks[i], text)
		}
	}
	if toks[0].Line != 1 || toks[1].Line != 2 || toks[2].Line != 4 {
		t.Fatalf("directive lines are %d, %d and %d, want 1, 2 and 4", toks[0].Line, toks[1].Line, toks[2].Line)
	}
}

func TestTokenizeMacroSpacingIsMeaningful(t *testing.T) {
	objectLike, err := Tokenize("#define F (x) (x)\n")
	if err != nil {
		t.Fatal(err)
	}
	functionLike, err := Tokenize("#define F(x) (x)\n")
	if err != nil {
		t.Fatal(err)
	}
	if objectLike[0].Text == functionLike[0].Text {
		t.Fatalf("object-like and function-like macros are both %q", objectLike[0].Text)
	}
}

func TestTokenizeLiteralsAndComments(t *testing.T) {
	src := `int f(void)
{
	// "not a string"
	char *s = "a \"b\" \\ c\n";
	char c = '\'';
	/* block "hidden" */
	int n = 0x1Fu + 3.5e-2;
	n <<= 1;
	return n;
}
`
	toks, err := Tokenize(src)
	if err != nil {
		t.Fatalf("tokenize: %v", err)
	}
	if tok, ok := findToken(toks, `"a \"b\" \\ c\n"`); !ok || tok.Kind != String {
		t.Fatalf("string literal lost its escapes: %+v", toks)
	}
	if tok, ok := findToken(toks, `'\''`); !ok || tok.Kind != Char {
		t.Fatalf("character literal lost its escape: %+v", toks)
	}
	for _, number := range []string{"0x1Fu", "3.5e-2"} {
		if tok, ok := findToken(toks, number); !ok || tok.Kind != Number {
			t.Fatalf("number %q is missing or not a Number: %+v", number, toks)
		}
	}
	if _, ok := findToken(toks, "<<="); !ok {
		t.Fatalf("operator <<= is missing: %+v", toks)
	}
	for _, tok := range toks {
		if strings.Contains(tok.Text, "not a string") || strings.Contains(tok.Text, "hidden") {
			t.Fatalf("comment text leaked into a token: %q", tok.Text)
		}
	}
}

func TestTokenizeWideLiteral(t *testing.T) {
	toks, err := Tokenize(`wchar_t *w = L"wide";`)
	if err != nil {
		t.Fatal(err)
	}
	if tok, ok := findToken(toks, `L"wide"`); !ok || tok.Kind != String {
		t.Fatalf("wide literal is not one String token: %+v", toks)
	}
}

func TestTokenizeOperators(t *testing.T) {
	src := "a->b += c && d != e || f >= g; x = (y[0] % 2) ? 1 : 0; p <<= 1; q >>= 2;"
	toks, err := Tokenize(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"->", "+=", "&&", "!=", "||", ">=", "?", ":", "[", "]", "<<=", ">>="} {
		if _, ok := findToken(toks, want); !ok {
			t.Fatalf("operator %q is missing: %+v", want, toks)
		}
	}
}

func TestTokenizeEmptySource(t *testing.T) {
	toks, err := Tokenize("")
	if err != nil {
		t.Fatalf("empty source: %v", err)
	}
	if len(toks) != 0 {
		t.Fatalf("empty source produced %d tokens", len(toks))
	}
}

func TestTokenizeMalformed(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"string", "int f(void) { char *s = \"abc; }", "unterminated string literal"},
		{"char", "int f(void) { char c = 'a; }", "unterminated character literal"},
		{"comment", "int f(void) { /* open", "unterminated block comment"},
		{"character", "int @ x;", "unexpected character"},
		{"hash mid line", "int x = 1; # define Y 2\n", "unexpected character"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Tokenize(tc.src); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error is %v, want one containing %q", err, tc.want)
			}
		})
	}
}

func TestTokenizeErrorLine(t *testing.T) {
	src := "int f(void)\n{\n\tchar *s = \"abc;\n}\n"
	_, err := Tokenize(src)
	if err == nil || !strings.Contains(err.Error(), "line 3: unterminated string literal") {
		t.Fatalf("error is %v, want line 3", err)
	}
}
