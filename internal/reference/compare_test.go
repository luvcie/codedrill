package reference

import (
	"math"
	"strings"
	"testing"
)

func TestCompareExactIgnoresLayoutAndComments(t *testing.T) {
	ref := "int f(void)\n{\n\treturn 0;\n}\n"
	student := "/* solution */\nint f(void) { return (0); } // done\n"
	res, err := Compare(ref, student)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !res.Exact || res.Score != 1 {
		t.Fatalf("expected an exact match, got %s", res.Summary())
	}
	if res.Missing != 0 || res.Extra != 0 || res.Matched != res.ReferenceTokens {
		t.Fatalf("exact match has differences: %+v", res)
	}
	if res.Percent() != 100 {
		t.Fatalf("exact match is %v%%, want 100", res.Percent())
	}
	if res.FirstDifferingLine != 0 {
		t.Fatalf("exact match points at line %d", res.FirstDifferingLine)
	}
}

func TestCompareCountsAndScore(t *testing.T) {
	cases := []struct {
		name                    string
		ref, student            string
		matched, missing, extra int
		score                   float64
	}{
		{"substitution", "int f ( void ) { return 1 ; }", "int f ( void ) { return 2 ; }", 9, 1, 1, 0.9},
		{"missing tail", "int f ( void ) { return 1 ; }", "int f ( void ) { }", 7, 3, 0, 0.7},
		{"extra tail", "a b", "a b c", 2, 0, 1, 2.0 / 3.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Compare(tc.ref, tc.student)
			if err != nil {
				t.Fatalf("compare: %v", err)
			}
			if res.Exact {
				t.Fatal("different sources compared as exact")
			}
			if res.Matched != tc.matched || res.Missing != tc.missing || res.Extra != tc.extra {
				t.Fatalf("got matched %d, missing %d, extra %d; want %d, %d, %d",
					res.Matched, res.Missing, res.Extra, tc.matched, tc.missing, tc.extra)
			}
			if math.Abs(res.Score-tc.score) > 1e-9 {
				t.Fatalf("score is %v, want %v", res.Score, tc.score)
			}
		})
	}
}

func TestCompareFirstDifferingLine(t *testing.T) {
	ref := "int f(void)\n{\n\treturn 0;\n}\n"
	student := "int f(void)\n{\n\t/* changed */\n\treturn 1;\n}\n"
	res, err := Compare(ref, student)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if res.FirstDifferingLine != 4 {
		t.Fatalf("first difference is on line %d, want 4", res.FirstDifferingLine)
	}
	if !strings.Contains(res.Summary(), "student line 4") {
		t.Fatalf("summary does not name the line: %s", res.Summary())
	}
}

func TestCompareDiffText(t *testing.T) {
	ref := "int f(void)\n{\n\treturn 0;\n}\n"
	student := "int f(void)\n{\n\treturn 1;\n}\n"
	res, err := Compare(ref, student)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	diff := res.DiffText()
	for _, want := range []string{"int f ( void )", "- 3: 0", "+ 3: 1"} {
		if !strings.Contains(diff, want) {
			t.Fatalf("diff is missing %q:\n%s", want, diff)
		}
	}
}

func TestDiffTextLabelsEachChangedSourceLine(t *testing.T) {
	res, err := Compare("int a;\nint b;\n"+strings.Repeat("x ", 12), "")
	if err != nil {
		t.Fatal(err)
	}
	diff := res.DiffText()
	for _, want := range []string{"- 1: int a ;", "- 2: int b ;", "- 3: "} {
		if !strings.Contains(diff, want) {
			t.Fatalf("diff lacks %q: %s", want, diff)
		}
	}
	if strings.Count(diff, "- 3: ") != 2 {
		t.Fatalf("wrapped token chunks lost their source line: %s", diff)
	}
}

func TestCompareEmptySources(t *testing.T) {
	res, err := Compare("", "")
	if err != nil || !res.Exact {
		t.Fatalf("two empty sources: %+v, %v", res, err)
	}
	if res.ReferenceTokens != 0 || res.StudentTokens != 0 || res.Score != 1 {
		t.Fatalf("empty sources report %+v", res)
	}

	res, err = Compare("", "int x;")
	if err != nil || res.Exact || res.Score != 0 || res.Extra != 3 || res.Missing != 0 {
		t.Fatalf("empty reference: %+v, %v", res, err)
	}
	if res.FirstDifferingLine != 1 {
		t.Fatalf("empty reference points at line %d, want 1", res.FirstDifferingLine)
	}

	res, err = Compare("int x;", "")
	if err != nil || res.Exact || res.Score != 0 || res.Missing != 3 || res.Extra != 0 {
		t.Fatalf("empty student: %+v, %v", res, err)
	}
	if res.FirstDifferingLine != 0 {
		t.Fatalf("empty student points at line %d, want 0", res.FirstDifferingLine)
	}

	res, err = Compare("// nothing here\n", "")
	if err != nil || !res.Exact {
		t.Fatalf("comment-only source: %+v, %v", res, err)
	}
}

func TestCompareNeverReportsHundredPercentWithoutExactMatch(t *testing.T) {
	ref := strings.Repeat("a ", 2000)
	student := strings.Repeat("a ", 1999) + "b"
	res, err := Compare(ref, student)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if res.Exact {
		t.Fatal("different sources compared as exact")
	}
	if res.Matched != 1999 {
		t.Fatalf("matched %d tokens, want 1999", res.Matched)
	}
	if p := res.Percent(); p >= 100 {
		t.Fatalf("non-exact match reports %v%%", p)
	}
}

func TestCompareReportsMalformedSource(t *testing.T) {
	_, err := Compare("int f(void) { return 0; }", "int f(void) { char *s = \"oops; }")
	if err == nil || !strings.Contains(err.Error(), "student: line 1: unterminated string literal") {
		t.Fatalf("student error is %v", err)
	}

	_, err = Compare("int f(void) { /* open", "int f(void) { return 0; }")
	if err == nil || !strings.Contains(err.Error(), "reference: line 1: unterminated block comment") {
		t.Fatalf("reference error is %v", err)
	}
}
