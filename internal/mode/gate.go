package mode

import "github.com/luvcie/codedrill/internal/compare"

// Verdict is the outcome of applying a mode's pass gates.
type Verdict struct {
	Passed bool
	Reason string
}

// Evaluate applies the required gates. Tests and the reference comparison are
// consulted only when their gate is required. A required exact-copy gate fails
// when the comparison is missing, errored, or did not match exactly, so a
// comparison problem can never pass a copy or recall run.
func Evaluate(opts Options, testsPassed bool, comparison *compare.Result, comparisonErr error) Verdict {
	opts = opts.Normalize()
	if err := opts.Validate(); err != nil {
		return Verdict{Reason: err.Error()}
	}
	if opts.RequireTests && !testsPassed {
		return Verdict{Reason: "tests did not pass"}
	}
	if opts.RequireExact {
		switch {
		case comparisonErr != nil:
			return Verdict{Reason: "reference comparison failed: " + comparisonErr.Error()}
		case comparison == nil:
			return Verdict{Reason: "reference comparison is unavailable"}
		case !comparison.Exact:
			return Verdict{Reason: "solution does not match the reference"}
		}
	}
	return Verdict{Passed: true}
}
