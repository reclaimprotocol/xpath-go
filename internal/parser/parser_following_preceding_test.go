package parser

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestParseFollowingAndPrecedingAxes(t *testing.T) {
	checks := []struct {
		expression string
		axis       types.XPathAxis
		nodeTest   string
		predicate  string
	}{
		{`//*[@id='a']/following::node()[1]`, types.AxisFollowing, `node()`, `1`},
		{`//*[@id='c']/preceding::*[last()]`, types.AxisPreceding, `*`, `last()`},
		{`//a/following::processing-instruction('next')[1]`, types.AxisFollowing, `processing-instruction('next')`, `1`},
	}
	for _, check := range checks {
		parsed, err := NewParser().Parse(check.expression)
		if err != nil {
			t.Fatalf("Parse(%q): %v", check.expression, err)
		}
		if len(parsed.Steps) == 0 {
			t.Fatalf("Parse(%q) returned no steps", check.expression)
		}
		step := parsed.Steps[len(parsed.Steps)-1]
		if step.Axis != check.axis || step.NodeTest != check.nodeTest || len(step.Predicates) != 1 || step.Predicates[0].Expression != check.predicate {
			t.Fatalf("Parse(%q) final step = %#v", check.expression, step)
		}
	}
}

func TestParseFollowingAndPrecedingUnionBranches(t *testing.T) {
	parsed, err := NewParser().Parse(`//a/following::* | //c/preceding::node()`)
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Union) != 2 {
		t.Fatalf("axis union branches = %#v", parsed)
	}
	wantAxes := []types.XPathAxis{types.AxisFollowing, types.AxisPreceding}
	for i, branch := range parsed.Union {
		if len(branch.Steps) == 0 || branch.Steps[len(branch.Steps)-1].Axis != wantAxes[i] {
			t.Fatalf("axis union branch %d = %#v", i, branch)
		}
	}
}
