package evaluator

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestUnifiedParserCompilesAxisNodeTestPredicates(t *testing.T) {
	expression, err := parseXPath(`child::span[@rank='keep']`)
	if err != nil {
		t.Fatal(err)
	}
	axis, ok := expression.(*AxisExpression)
	if !ok {
		t.Fatalf("axis expression = %T, want *AxisExpression", expression)
	}
	if axis.NodeTest != "span" || len(axis.Predicates) != 1 {
		t.Fatalf("axis expression = %#v, want typed span predicate", axis)
	}
	if _, ok := axis.Predicates[0].(*ComparisonExpression); !ok {
		t.Fatalf("axis predicate = %T, want *ComparisonExpression", axis.Predicates[0])
	}

	parent := &types.Node{Type: types.ElementNode, Name: "div"}
	keep := &types.Node{Type: types.ElementNode, Name: "span", Parent: parent, Attributes: map[string]string{"rank": "keep"}, AttributeOrder: []string{"rank"}}
	skip := &types.Node{Type: types.ElementNode, Name: "span", Parent: parent, Attributes: map[string]string{"rank": "skip"}, AttributeOrder: []string{"rank"}}
	parent.Children = []*types.Node{keep, skip}
	value := evaluateXPathValue(axis, parent, NewEvaluator())
	if value.kind != nodeSetXPathValue || len(value.nodes) != 1 || value.nodes[0] != keep {
		t.Fatalf("typed axis predicate result = %#v, want only the keep span", value)
	}
}

func TestCompiledAxisExpressionRetainsFollowingPath(t *testing.T) {
	program, err := Compile(`//div[child::span[@id='x']/b[@class='keep']]`)
	if err != nil {
		t.Fatal(err)
	}
	predicate := finalPathPredicate(t, program)
	axis, ok := predicate.(*AxisExpression)
	if !ok {
		t.Fatalf("compiled predicate = %T, want *AxisExpression", predicate)
	}
	if axis.NodeTest != "span" || len(axis.Predicates) != 1 || len(axis.Following) != 1 {
		t.Fatalf("axis AST = %#v, want typed span predicate and b tail", axis)
	}
	if axis.Following[0].Name != "b" || len(axis.Following[0].Predicates) != 1 {
		t.Fatalf("axis tail = %#v, want typed b predicate", axis.Following)
	}

	results, err := NewEvaluator().EvaluateProgramWithDocument(program, `<div><span id=x><b class=keep>yes</b></span></div><div><span id=x><b class=skip>no</b></span></div><div><span id=y><b class=keep>no</b></span></div>`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].TextContent != "yes" {
		t.Fatalf("axis tail results = %#v, want only the matching div", results)
	}
}

func TestCompiledAxisExpressionRetainsNestedFollowingPath(t *testing.T) {
	program, err := Compile(`//article[child::section[child::span[@id='x']/b]]`)
	if err != nil {
		t.Fatal(err)
	}
	results, err := NewEvaluator().EvaluateProgramWithDocument(program, `<article><section><span id=x><b>yes</b></span></section></article><article><section><span id=x></span></section></article>`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].TextContent != "yes" {
		t.Fatalf("nested axis tail results = %#v, want only matching article", results)
	}
}
