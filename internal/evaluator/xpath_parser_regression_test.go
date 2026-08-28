package evaluator

import (
	"strings"
	"testing"
)

func TestUnifiedParserSupportsMixedExplicitAxisTails(t *testing.T) {
	program, err := Compile(`//section/child::span[@title='[東京]']/following-sibling::a[1] | //aside/descendant::a[@id='other']`)
	if err != nil {
		t.Fatal(err)
	}
	results, err := NewEvaluator().EvaluateProgramWithDocument(program, `<section><span title="[東京]">x</span><a id=first>first</a><a>later</a></section><aside><div><a id=other>other</a></div></aside>`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].TextContent != "first" || results[1].TextContent != "other" {
		t.Fatalf("mixed-axis result = %#v, want first and other links", results)
	}
}

func TestUnifiedParserReportsSourceOffsetsAndRejectsMalformedSyntax(t *testing.T) {
	for _, expression := range []string{
		`//item[1.2.3]`,
		`//item[@name='unterminated]`,
		`//item[@name=]`,
		`//item |`,
		`//item[string('a', 'b')]`,
	} {
		_, err := Compile(expression)
		if err == nil {
			t.Errorf("Compile(%q) succeeded, want syntax error", expression)
			continue
		}
		if !strings.Contains(err.Error(), "byte ") {
			t.Errorf("Compile(%q) error %q lacks source offset", expression, err)
		}
	}
}

func TestUnifiedParserHandlesUnicodeAndMultiplicationPrecedence(t *testing.T) {
	program, err := Compile(`//item[@title='café — 東京' and 1+2*3=7]`)
	if err != nil {
		t.Fatal(err)
	}
	results, err := NewEvaluator().EvaluateProgramWithDocument(program, `<item title="café — 東京">ok</item><item title="skip">no</item>`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].TextContent != "ok" {
		t.Fatalf("Unicode/precedence result = %#v", results)
	}
}

func TestUnifiedParserPersistsNestedSourceSpans(t *testing.T) {
	const source = `//article[count(.//span[@title='x'])=1]`
	program, err := Compile(source)
	if err != nil {
		t.Fatal(err)
	}
	outer, ok := program.root.(*PathExpression)
	if !ok || outer.Span != (SourceSpan{Start: 0, End: len(source)}) {
		t.Fatalf("outer path span = %#v, want whole source", outer)
	}
	predicate := outer.Steps[0].Predicates[0].(*ComparisonExpression)
	function := predicate.Left.(*FunctionExpression).Function
	functionStart := strings.Index(source, "count(")
	functionEnd := strings.Index(source, ")=1") + 1
	if function.Span != (SourceSpan{Start: functionStart, End: functionEnd}) {
		t.Fatalf("function span = %#v, want [%d,%d)", function.Span, functionStart, functionEnd)
	}
	inner := function.Arguments[0].(*PathExpression)
	innerStart := strings.Index(source, ".//span")
	innerEnd := strings.Index(source, ")=1")
	if inner.Span != (SourceSpan{Start: innerStart, End: innerEnd}) {
		t.Fatalf("nested path span = %#v, want [%d,%d)", inner.Span, innerStart, innerEnd)
	}
	stepStart := strings.Index(source, "span")
	if inner.Steps[1].Span.Start != stepStart || inner.Steps[1].Span.End != innerEnd {
		t.Fatalf("nested step span = %#v, want [%d,%d)", inner.Steps[1].Span, stepStart, innerEnd)
	}
}

func TestUnifiedParserPersistsAxisAndOperatorSourceSpans(t *testing.T) {
	const source = `child::span[@id='x']/following-sibling::a[1+2*3=7]`
	expression, err := parseXPath(source)
	if err != nil {
		t.Fatal(err)
	}
	axis := expression.(*AxisExpression)
	if axis.Span != (SourceSpan{Start: 0, End: len(source)}) {
		t.Fatalf("axis span = %#v, want whole source", axis.Span)
	}
	comparison := axis.Following[0].Predicates[0].(*ComparisonExpression)
	if comparison.Span != (SourceSpan{Start: strings.Index(source, "1+2"), End: len(source) - 1}) {
		t.Fatalf("comparison span = %#v", comparison.Span)
	}
	addition := comparison.Left.(*ArithmeticExpression)
	if addition.Span != (SourceSpan{Start: strings.Index(source, "1+2"), End: strings.Index(source, "=7")}) {
		t.Fatalf("additive span = %#v", addition.Span)
	}
	multiply := addition.Right.(*ArithmeticExpression)
	if multiply.Span != (SourceSpan{Start: strings.Index(source, "2*3"), End: strings.Index(source, "=7")}) {
		t.Fatalf("multiplicative span = %#v", multiply.Span)
	}
}

func TestParenthesizedNodeSetPredicateAndPathSuffix(t *testing.T) {
	const content = `<div><span class="preferred_name">First</span></div>` +
		`<div><span class="preferred_name">Second</span></div>` +
		`<span class="surname">Rivera</span>`

	for expression, want := range map[string]string{
		`(//span[@class='preferred_name'])[1]/text()`: "First",
		`(//span[@class='surname'])[1]/text()`:        "Rivera",
	} {
		program, err := Compile(expression)
		if err != nil {
			t.Fatalf("Compile(%q): %v", expression, err)
		}
		results, err := NewEvaluator().EvaluateProgramWithDocument(program, content, nil)
		if err != nil {
			t.Fatalf("Evaluate(%q): %v", expression, err)
		}
		if len(results) != 1 || results[0].TextContent != want {
			t.Fatalf("Evaluate(%q) = %#v, want one text node %q", expression, results, want)
		}
	}
}

func TestParenthesizedPredicateUsesWholeNodeSetPosition(t *testing.T) {
	const content = `<div><span>A</span><span>B</span></div><div><span>C</span></div>`
	program, err := Compile(`(//span)[1]/text()`)
	if err != nil {
		t.Fatal(err)
	}
	results, err := NewEvaluator().EvaluateProgramWithDocument(program, content, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].TextContent != "A" {
		t.Fatalf("result = %#v, want only A", results)
	}
}
