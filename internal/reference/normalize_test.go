package reference

import (
	"reflect"
	"strings"
	"testing"
)

func normalizedText(t *testing.T, src string) string {
	t.Helper()
	toks, err := Tokenize(src)
	if err != nil {
		t.Fatalf("tokenize %q: %v", src, err)
	}
	return joinTexts(Normalize(toks))
}

func joinTexts(toks []Token) string {
	parts := make([]string, len(toks))
	for i, tok := range toks {
		parts[i] = tok.Text
	}
	return strings.Join(parts, " ")
}

func TestNormalizeReturnExpressions(t *testing.T) {
	cases := []struct {
		name         string
		ref, student string
	}{
		{"atom", "int f(void) { return (0); }", "int f(void) { return 0; }"},
		{"nested", "int f(int x) { return ((x + 1)); }", "int f(int x) { return x + 1; }"},
		{"if body", "int f(int x) { if (x) return (1); return 0; }", "int f(int x) { if (x) return 1; return 0; }"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Compare(tc.ref, tc.student)
			if err != nil {
				t.Fatalf("compare: %v", err)
			}
			if !res.Exact {
				t.Fatalf("expected an exact match after normalization, got %s", res.Summary())
			}
		})
	}
}

func TestNormalizeKeepsStructuralParentheses(t *testing.T) {
	cases := []struct {
		name         string
		ref, student string
	}{
		{"product", "int f(int a, int b, int c) { return (a + b) * c; }", "int f(int a, int b, int c) { return a + b * c; }"},
		{"operand", "int f(int a) { return (a) + 1; }", "int f(int a) { return a + 1; }"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Compare(tc.ref, tc.student)
			if err != nil {
				t.Fatalf("compare: %v", err)
			}
			if res.Exact {
				t.Fatalf("normalization removed parentheses that shape the expression")
			}
		})
	}
}

func TestNormalizeKeepsCastParentheses(t *testing.T) {
	got := normalizedText(t, "int f(int x) { return (int)x; }")
	if !strings.Contains(got, "return ( int ) x ;") {
		t.Fatalf("cast parentheses were changed: %s", got)
	}
}

func TestNormalizeIsIdempotentAndCopies(t *testing.T) {
	toks, err := Tokenize("int f(void) { return ((0)); }")
	if err != nil {
		t.Fatal(err)
	}
	once := Normalize(toks)
	twice := Normalize(once)
	if !reflect.DeepEqual(once, twice) {
		t.Fatalf("normalization is not idempotent:\n%+v\n%+v", once, twice)
	}
	if _, ok := findToken(toks, "("); !ok {
		t.Fatal("Normalize modified its input tokens")
	}
}
