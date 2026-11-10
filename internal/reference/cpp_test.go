package reference

import (
	"slices"
	"strings"
	"testing"
)

func TestCPPTokenSpelling(t *testing.T) {
	cases := []struct {
		name, source string
		want         []string
	}{
		{"contextual identifiers", "int module; int import;", []string{"int", "module", ";", "int", "import", ";"}},
		{"operators", "a::b ->* p .* q >> 2 >>= 1 << 3 <<= 2 <=> c -> d ... e += f && g != h;", []string{"a", "::", "b", "->*", "p", ".*", "q", ">>", "2", ">>=", "1", "<<", "3", "<<=", "2", "<=>", "c", "->", "d", "...", "e", "+=", "f", "&&", "g", "!=", "h", ";"}},
		{"templates", "std::vector<std::vector<int>> v; A<::B> b;", []string{"std", "::", "vector", "<", "std", "::", "vector", "<", "int", ">>", "v", ";", "A", "<", "::", "B", ">", "b", ";"}},
		{"numbers", "0 42ULL 0xffu 0b1010 1'000 0xA'B .5 1.2e-3F 0x1.fp+2 12_km", []string{"0", "42ULL", "0xffu", "0b1010", "1'000", "0xA'B", ".5", "1.2e-3F", "0x1.fp+2", "12_km"}},
		{"literals", `u8"// /* \" \\ \n" L'\'' U'\u0041' "x"_tag R"custom(a " // /* )other" b)custom"`, []string{`u8"// /* \" \\ \n"`, `L'\''`, `U'\u0041'`, `"x"_tag`, `R"custom(a " // /* )other" b)custom"`}},
		{"escapes", `"\0\123\xff\u0041\U0001F600\a\b\f\n\r\t\v\?"`, []string{`"\0\123\xff\u0041\U0001F600\a\b\f\n\r\t\v\?"`}},
		{"raw prefixes", `u8R"(a)" uR"x(b)x" UR"(c)" LR"(d)"`, []string{`u8R"(a)"`, `uR"x(b)x"`, `UR"(c)"`, `LR"(d)"`}},
		{"spliced identifier", "long na\\\nme; /\\\n/ hidden\\\nstill hidden\nint x;", []string{"long", "name", ";", "int", "x", ";"}},
		{"macro continuation", "# define ADD(x) \\\n ((x)+1) // note\nint x;", []string{"#define ADD(x) ((x)+1)", "int", "x", ";"}},
		{"header name", "#include <dir//header.hpp>\n", []string{"#include <dir//header.hpp>"}},
		{"header query", "#if __has_include(<dir//header.hpp>)\n", []string{"#if __has_include(<dir//header.hpp>)"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tokens, err := TokenizeCPP(tc.source)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, tok := range tokens {
				got = append(got, tok.Text)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("tokens = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCPPPhysicalLinesAndRawSplices(t *testing.T) {
	src := "#define F(x) \\\r\n (x)\r\n/* comment\n */ auto s = R\"tag(first\\\nsecond\n)tag\";\nint na\\\nme;"
	tokens, err := TokenizeCPP(src)
	if err != nil {
		t.Fatal(err)
	}
	for text, line := range map[string]int{
		"#define F(x) (x)": 1, "auto": 4,
		"R\"tag(first\\\nsecond\n)tag\"": 4,
		"int":                            7, "name": 7,
	} {
		tok, ok := findToken(tokens, text)
		if !ok || tok.Line != line {
			t.Fatalf("%q: %+v, found %v, want line %d", text, tok, ok, line)
		}
	}
}

func TestCPPComparisonAndDiff(t *testing.T) {
	ref := "#include <vector>\nstd::vector<int> f() {\nreturn {1, 2};\n}"
	student := "# include <vector> // header\n/* before */ std :: vector < int > f ( ) {\nreturn {1, 2}; }"
	exact, err := CompareLanguage("cpp", ref, student)
	if err != nil || !exact.Exact || exact.Percent() != 100 {
		t.Fatalf("layout comparison: %+v, %v", exact, err)
	}
	student = strings.Replace(student, "1, 2", "1, 3", 1)
	partial, err := CompareLanguage("cpp", ref, student)
	if err != nil || partial.Exact || partial.Percent() <= 0 || partial.Percent() >= 100 {
		t.Fatalf("partial comparison: %+v, %v", partial, err)
	}
	if diff := partial.DiffText(); !strings.Contains(diff, "- 3: 2") || !strings.Contains(diff, "+ 3: 3") {
		t.Fatalf("line-labelled diff: %s", diff)
	}
}

func TestCPPAvoidsFalseExactMatches(t *testing.T) {
	pairs := [][2]string{
		{"decltype(auto) f() { return x; }", "decltype(auto) f() { return (x); }"},
		{"x >> y;", "x > > y;"},
		{"a->*p;", "a-> *p;"},
		{"a.*p;", "a. *p;"},
		{"A::b;", "A: :b;"},
		{"a <=> b;", "a <= > b;"},
		{`"x"_tag`, `"x" _tag`},
		{`u8"x"`, `u8 "x"`},
		{"1e+2", "1e +2"},
		{"#define F(x) x\n", "#define F (x) x\n"},
		{"#define F\\\n(x) x\n", "#define F \\\n(x) x\n"},
		{"R\"(a\\\nb)\"", "R\"(ab)\""},
		{`"//comment"`, `""`},
		{"#if __has_include(<a//b>)\n", "#if __has_include(<a//c>)\n"},
		{"int n = __LINE__;", "\nint n = __LINE__;"},
	}
	for _, pair := range pairs {
		res, err := CompareLanguage("cpp", pair[0], pair[1])
		if err != nil || res.Exact || res.Percent() >= 100 {
			t.Errorf("%q vs %q: exact=%v, err=%v", pair[0], pair[1], res.Exact, err)
		}
	}
}

func TestCPPLexicalErrorsFailClosed(t *testing.T) {
	for _, source := range []string{
		`"unterminated`, `'x`, `''`, `/* unfinished`, `"\x"`, `"\u12"`, `"\q"`,
		`R"(unfinished`, `R"bad delimiter(x)bad delimiter"`,
		`R"12345678901234567(x)12345678901234567"`,
		"R\"bad\\\n(x)bad\"", `int @x;`, `int \u0041;`, "int \xff;",
		"int x; #define X 1", "import <a//b>;", "export module X;", "#define S(x) #x\nS(a+b)",
		"#include <unfinished\n", "const char* s = \"line\nbreak\";",
	} {
		t.Run(source, func(t *testing.T) {
			if tokens, err := TokenizeCPP(source); err == nil {
				t.Fatalf("unsupported source accepted: %+v", tokens)
			}
			for _, pair := range [][2]string{{source, "int x;"}, {"int x;", source}, {source, source}} {
				res, err := CompareLanguage("cpp", pair[0], pair[1])
				if err == nil || res.Exact {
					t.Fatalf("lexical failure did not fail closed: %+v, %v", res, err)
				}
			}
		})
	}
	_, err := TokenizeCPP("\n\nR\"(unfinished")
	if err == nil || !strings.Contains(err.Error(), "line 3:") {
		t.Fatalf("error line lost: %v", err)
	}
	// Partial but lexically complete code still supports useful live scores.
	res, err := CompareLanguage("cpp", "class X { int f(); };", "class X {")
	if err != nil || res.Exact || res.Matched == 0 {
		t.Fatalf("incomplete source: %+v, %v", res, err)
	}
}

func TestCPPResourceLimitAndAliases(t *testing.T) {
	for _, language := range []string{"cpp", "C++", " cc ", "cxx"} {
		res, err := CompareLanguage(language, "int x;", "int /* note */ x;")
		if err != nil || !res.Exact {
			t.Fatalf("%s: %+v, %v", language, res, err)
		}
	}
	large := strings.Repeat("x ", 2001)
	for _, language := range []string{"c", "cpp"} {
		res, err := CompareLanguage(language, large, large)
		if err == nil || res.Exact || !strings.Contains(err.Error(), "too large") {
			t.Fatalf("%s resource limit: %+v, %v", language, res, err)
		}
	}
}
