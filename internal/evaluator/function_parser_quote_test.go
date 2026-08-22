package evaluator

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestUnifiedParserPathPredicateHonorsQuotedBrackets(t *testing.T) {
	tests := []struct {
		expression string
		want       string
	}{
		{`count(.//span[@title=']'])`, `count(.//span[@title = ']'])`},
		{`string(.//span[@title='[x]'])`, `string(.//span[@title = '[x]'])`},
		{`count(.//span[@title="["][@data="]"])`, `count(.//span[@title = '['][@data = ']'])`},
		{`count(../span[@title=']'])`, `count(../span[@title = ']'])`},
		{`count(.//span/..[@title=']'])`, `count(.//span/..[@title = ']'])`},
	}

	for _, test := range tests {
		t.Run(test.expression, func(t *testing.T) {
			expression, err := parseXPath(test.expression)
			if err != nil {
				t.Fatalf("Parse(%q): %v", test.expression, err)
			}
			if got := expression.String(); got != test.want {
				t.Errorf("Parse(%q).String() = %q, want %q", test.expression, got, test.want)
			}
		})
	}
}

func TestUnifiedParserPathPredicateRejectsUnterminatedLiteral(t *testing.T) {
	if _, err := parseXPath(`count(.//span[@title='missing])`); err == nil {
		t.Fatal("Parse accepted an unterminated literal in a nested path predicate")
	}
}

func TestUnifiedParserRelativeParentPath(t *testing.T) {
	path, err := parseXPath(`../span`)
	if err != nil {
		t.Fatal(err)
	}
	parent := &types.Node{Type: types.ElementNode, Name: "section"}
	context := &types.Node{Type: types.ElementNode, Name: "div", Parent: parent}
	span := &types.Node{Type: types.ElementNode, Name: "span", Parent: parent}
	parent.Children = []*types.Node{context, span}
	value := evaluateXPathValue(path, context, NewEvaluator())
	if value.kind != nodeSetXPathValue || len(value.nodes) != 1 || value.nodes[0] != span {
		t.Fatalf("../span = %#v, want the parent's span child", value)
	}
}

func TestUnifiedParserAppliesPredicatesToAbbreviatedSteps(t *testing.T) {
	self, err := parseXPath(`.[@id='keep']`)
	if err != nil {
		t.Fatal(err)
	}
	node := &types.Node{Type: types.ElementNode, Name: "div", Attributes: map[string]string{"id": "keep"}, AttributeOrder: []string{"id"}}
	value := evaluateXPathValue(self, node, NewEvaluator())
	if value.kind != nodeSetXPathValue || len(value.nodes) != 1 || value.nodes[0] != node {
		t.Fatalf("self predicate = %#v, want the context node", value)
	}

	attribute, err := parseXPath(`@id[.='keep']`)
	if err != nil {
		t.Fatal(err)
	}
	value = evaluateXPathValue(attribute, node, NewEvaluator())
	if value.kind != nodeSetXPathValue || len(value.nodes) != 1 || value.nodes[0].Name != "id" {
		t.Fatalf("attribute predicate = %#v, want id attribute", value)
	}
}
