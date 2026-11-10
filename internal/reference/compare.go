package reference

import (
	"fmt"
	"math"
	"strings"
)

// Op is what happened to a token in the diff.
type Op int

const (
	Equal   Op = iota
	Missing    // in the reference, absent from the student
	Extra      // in the student, absent from the reference
)

// DiffOp is one step of the token diff. Token belongs to the reference for
// Missing and to the student for Equal and Extra.
type DiffOp struct {
	Op    Op
	Token Token
}

// Result is the outcome of comparing a student file with a reference.
type Result struct {
	Exact           bool
	Score           float64
	ReferenceTokens int
	StudentTokens   int
	Matched         int
	Missing         int
	Extra           int
	// FirstDifferingLine is the student line of the first extra token, or 0
	// when the student has no token outside the match.
	FirstDifferingLine int
	Ops                []DiffOp
}

// maxCells bounds the LCS table. Drill sources sit far below this.
const maxCells = 4_000_000

// Compare tokenizes both sources, normalizes redundant return parentheses and
// scores their token sequences. Either source failing to tokenize is an error,
// not a zero score.
func Compare(referenceSrc, studentSrc string) (Result, error) {
	return CompareLanguage("c", referenceSrc, studentSrc)
}

// SupportedLanguage reports whether the shared comparison pipeline supports language.
func SupportedLanguage(language string) bool {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "c", "cpp", "c++", "cc", "cxx":
		return true
	}
	return false
}

// CompareLanguage selects lexical rules while sharing scoring and diff output.
// C++ retains return parentheses: they can change decltype(auto) deduction.
func CompareLanguage(language, referenceSrc, studentSrc string) (Result, error) {
	language = strings.ToLower(strings.TrimSpace(language))
	if !SupportedLanguage(language) {
		return Result{}, fmt.Errorf("reference comparison is not supported for language %q", language)
	}
	tokenize := TokenizeCPP
	if language == "c" {
		tokenize = Tokenize
	}
	ref, err := tokenize(referenceSrc)
	if err != nil {
		return Result{}, fmt.Errorf("reference: %w", err)
	}
	stu, err := tokenize(studentSrc)
	if err != nil {
		return Result{}, fmt.Errorf("student: %w", err)
	}
	if language == "c" {
		ref, stu = Normalize(ref), Normalize(stu)
	}
	if len(ref) > 0 && len(stu) > maxCells/len(ref) {
		return Result{}, fmt.Errorf("sources too large to compare: %d reference tokens, %d student tokens", len(ref), len(stu))
	}
	return compareTokens(ref, stu), nil
}

func compareTokens(ref, stu []Token) Result {
	res := Result{ReferenceTokens: len(ref), StudentTokens: len(stu)}
	width := len(stu) + 1
	lcs := make([]int32, (len(ref)+1)*width)
	for i := 1; i <= len(ref); i++ {
		for j := 1; j <= len(stu); j++ {
			if sameToken(ref[i-1], stu[j-1]) {
				lcs[i*width+j] = lcs[(i-1)*width+j-1] + 1
				continue
			}
			down, left := lcs[(i-1)*width+j], lcs[i*width+j-1]
			if down >= left {
				lcs[i*width+j] = down
			} else {
				lcs[i*width+j] = left
			}
		}
	}

	res.Matched = int(lcs[len(ref)*width+len(stu)])
	res.Missing = len(ref) - res.Matched
	res.Extra = len(stu) - res.Matched
	res.Exact = res.Missing == 0 && res.Extra == 0
	if total := max(len(ref), len(stu)); total > 0 {
		res.Score = float64(res.Matched) / float64(total)
	} else {
		res.Score = 1
	}
	// A score of exactly 1 means the sequences are equal, so only an exact
	// match may report 100%.
	if !res.Exact && res.Score >= 1 {
		res.Score = 0.9999
	}

	res.Ops = diffTokens(ref, stu, lcs, width)
	for _, op := range res.Ops {
		if op.Op == Extra {
			res.FirstDifferingLine = op.Token.Line
			break
		}
	}
	return res
}

// diffTokens walks the LCS table backwards, then reverses the result so the
// reference-only tokens come before student-only tokens at a substitution.
func diffTokens(ref, stu []Token, lcs []int32, width int) []DiffOp {
	i, j := len(ref), len(stu)
	ops := make([]DiffOp, 0, len(ref)+len(stu))
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && sameToken(ref[i-1], stu[j-1]):
			ops = append(ops, DiffOp{Op: Equal, Token: stu[j-1]})
			i--
			j--
		case j > 0 && (i == 0 || lcs[i*width+j-1] >= lcs[(i-1)*width+j]):
			ops = append(ops, DiffOp{Op: Extra, Token: stu[j-1]})
			j--
		default:
			ops = append(ops, DiffOp{Op: Missing, Token: ref[i-1]})
			i--
		}
	}
	for l, r := 0, len(ops)-1; l < r; l, r = l+1, r-1 {
		ops[l], ops[r] = ops[r], ops[l]
	}
	return ops
}

func sameToken(a, b Token) bool { return a.Kind == b.Kind && a.Text == b.Text }

// Percent is the similarity rounded to one decimal. It stays below 100 unless
// the comparison is exact.
func (r Result) Percent() float64 {
	return Percent(r.Score, r.Exact)
}

// Percent rounds a score to one decimal, reserving 100 for exact matches.
func Percent(score float64, exact bool) float64 {
	if exact {
		return 100
	}
	p := math.Round(score*1000) / 10
	if p >= 100 {
		p = 99.9
	}
	return p
}

// Summary is a one-line description of the result, for screens and logs.
func (r Result) Summary() string {
	if r.Exact {
		return fmt.Sprintf("exact match: %d tokens", r.ReferenceTokens)
	}
	line := "no student line to point at"
	if r.FirstDifferingLine > 0 {
		line = fmt.Sprintf("first difference at student line %d", r.FirstDifferingLine)
	}
	return fmt.Sprintf("%.1f%% similar: matched %d of %d reference tokens (%d missing, %d extra), %s",
		r.Percent(), r.Matched, r.ReferenceTokens, r.Missing, r.Extra, line)
}

// DiffText renders the token diff for this result.
func (r Result) DiffText() string { return FormatDiff(r.Ops) }

const (
	diffChunk   = 10
	diffContext = 4
)

// FormatDiff renders a token-level diff. Missing tokens are prefixed with "-"
// and carry their reference line, extra tokens with "+" and their student
// line. Equal runs longer than eight tokens are shortened.
func FormatDiff(ops []DiffOp) string {
	var b strings.Builder
	for i := 0; i < len(ops); {
		j := i
		for j < len(ops) && ops[j].Op == ops[i].Op {
			j++
		}
		group := ops[i:j]
		switch group[0].Op {
		case Equal:
			if len(group) > 2*diffContext {
				fmt.Fprintf(&b, "  %s ... %s\n", joinTokens(group[:diffContext]), joinTokens(group[len(group)-diffContext:]))
			} else {
				fmt.Fprintf(&b, "  %s\n", joinTokens(group))
			}
		default:
			marker := "- "
			if group[0].Op == Extra {
				marker = "+ "
			}
			for k := 0; k < len(group); {
				end := k + 1
				for end < len(group) && end < k+diffChunk && group[end].Token.Line == group[k].Token.Line {
					end++
				}
				fmt.Fprintf(&b, "%s%d: %s\n", marker, group[k].Token.Line, joinTokens(group[k:end]))
				k = end
			}
		}
		i = j
	}
	return b.String()
}

func joinTokens(group []DiffOp) string {
	parts := make([]string, len(group))
	for i, op := range group {
		parts[i] = op.Token.Text
	}
	return strings.Join(parts, " ")
}
