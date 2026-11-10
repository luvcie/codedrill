package mode

import (
	"errors"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/compare"
)

func TestEvaluateGates(t *testing.T) {
	exact := &compare.Result{Exact: true}
	mismatch := &compare.Result{Exact: false}
	both := Options{CheckReference: true, RequireExact: true, RequireTests: true}
	cases := []struct {
		name          string
		opts          Options
		testsPassed   bool
		comparison    *compare.Result
		comparisonErr error
		pass          bool
		reason        string
	}{
		{"study passes", Study(), true, exact, nil, true, ""},
		{"study needs tests", Study(), false, exact, nil, false, "tests did not pass"},
		{"copy ignores tests", Copy(), false, exact, nil, true, ""},
		{"copy rejects mismatch", Copy(), true, mismatch, nil, false, "does not match"},
		{"copy fails on missing result", Copy(), true, nil, nil, false, "unavailable"},
		{"copy fails on comparison error", Copy(), true, nil, errors.New("no reference solution"), false, "no reference solution"},
		{"recall ignores tests", Recall(), false, exact, nil, true, ""},
		{"exam ignores comparison", Exam(), true, nil, errors.New("boom"), true, ""},
		{"exam ignores mismatch", Exam(), true, mismatch, nil, true, ""},
		{"both pass", both, true, exact, nil, true, ""},
		{"both need tests", both, false, exact, nil, false, "tests"},
		{"both need exact", both, true, mismatch, nil, false, "match"},
		{"invalid options fail", Options{}, true, exact, nil, false, "pass gate"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Evaluate(tc.opts, tc.testsPassed, tc.comparison, tc.comparisonErr)
			if got.Passed != tc.pass {
				t.Fatalf("passed is %v, want %v (%s)", got.Passed, tc.pass, got.Reason)
			}
			if tc.reason != "" && !strings.Contains(got.Reason, tc.reason) {
				t.Fatalf("reason is %q, want one containing %q", got.Reason, tc.reason)
			}
			if got.Passed && got.Reason != "" {
				t.Fatalf("passing verdict has reason %q", got.Reason)
			}
		})
	}
}

func TestEvaluateNormalizesReferenceOptions(t *testing.T) {
	opts := Options{ShowReference: true, RequireExact: true, RequireTests: true}
	if got := Evaluate(opts, true, &compare.Result{Exact: true}, nil); !got.Passed {
		t.Fatalf("tests gate should pass: %+v", got)
	}
}
