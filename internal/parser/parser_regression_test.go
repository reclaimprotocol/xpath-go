package parser

import (
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseShorthandSelfAndParentUseNodeTests(t *testing.T) {
	tests := []struct {
		expression string
		axis       types.XPathAxis
	}{
		{expression: `.`, axis: types.AxisSelf},
		{expression: `..`, axis: types.AxisParent},
	}
	for _, test := range tests {
		parsed, err := NewParser().Parse(test.expression)
		if err != nil {
			t.Fatalf("Parse(%q): %v", test.expression, err)
		}
		if len(parsed.Steps) != 1 {
			t.Fatalf("Parse(%q) returned %d steps, want one", test.expression, len(parsed.Steps))
		}
		step := parsed.Steps[0]
		if step.Axis != test.axis || step.NodeTest != "node()" {
			t.Errorf("Parse(%q) = axis %q, node test %q; want axis %q and node()", test.expression, step.Axis, step.NodeTest, test.axis)
		}
	}
}

func TestParseRejectsMalformedDelimitersAndOperators(t *testing.T) {
	for _, expression := range []string{
		`//div[`,
		`//div[@id='unterminated]`,
		`//div[(1 + 2]`,
		`//div[@id = ]`,
		`//div |`,
		`//div//`,
		`//div[1 2]`,
	} {
		if _, err := NewParser().Parse(expression); err == nil {
			t.Errorf("Parse(%q) succeeded, want malformed-expression error", expression)
		}
	}
}

func TestParseSupportsUnicodeInPredicateLiterals(t *testing.T) {
	parsed, err := NewParser().Parse(`//article[@title='café — 東京']`)
	if err != nil {
		t.Fatalf("Parse Unicode predicate: %v", err)
	}
	if len(parsed.Steps) != 2 || len(parsed.Steps[1].Predicates) != 1 {
		t.Fatalf("unexpected parse result: %#v", parsed)
	}
	if !strings.Contains(parsed.Steps[1].Predicates[0].Expression, "東京") {
		t.Fatalf("Unicode literal was lost: %q", parsed.Steps[1].Predicates[0].Expression)
	}
}

// FuzzParseXPath is intentionally safe to run as ordinary seed coverage:
// malformed input is expected to return an error, but must never panic or
// leave the parser in a state that affects a later parse.
func FuzzParseXPath(f *testing.F) {
	for _, seed := range []string{
		`//div[@id='x']`,
		`/html/body/main | //article[position()=last()]`,
		`//*[@data-value="a|b[c]"]`,
		`//a[not(@href) and number(@rank) >= 1]`,
		`//div[(1 + 2) * 3 div 4]`,
		`//東京[@title='東京']`,
		`//div[`,
		`//div[@x='unterminated]`,
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, expression string) {
		p := NewParser()
		_, _ = p.Parse(expression)
		// Parser instances are reusable; a failed parse must not poison the
		// token stream used by a subsequent valid expression.
		if parsed, err := p.Parse(`//stable`); err != nil || parsed == nil || len(parsed.Steps) != 2 {
			t.Fatalf("parser reuse after %q: parsed=%#v err=%v", expression, parsed, err)
		}
	})
}
