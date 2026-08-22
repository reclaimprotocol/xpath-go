package evaluator

import (
	"strings"
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func typedConversionContext() *types.Node {
	node := &types.Node{
		Type:        types.ElementNode,
		Name:        "div",
		TextContent: " \t01\n",
		Attributes: map[string]string{
			"code":  "01",
			"plain": "1",
			"empty": "",
		},
	}
	first := &types.Node{Type: types.ElementNode, Name: "z", TextContent: "first", Parent: node}
	second := &types.Node{Type: types.ElementNode, Name: "a", TextContent: "second", Parent: node}
	node.Children = []*types.Node{first, second}
	return node
}

func evaluateTypedExpression(t *testing.T, node *types.Node, expression string) string {
	t.Helper()
	parsed, err := parseXPath(expression)
	if err != nil {
		t.Fatalf("Parse(%q): %v", expression, err)
	}
	return evaluateXPathValue(parsed, node, NewEvaluator()).legacyString()
}

func TestUnifiedParserPreservesTypedConversionStructure(t *testing.T) {
	checks := []struct {
		expression string
		name       string
		arguments  int
	}{
		{`string()`, "string", 0},
		{`number(@code)`, "number", 1},
		{`boolean(string(@empty))`, "boolean", 1},
	}

	for _, check := range checks {
		expression, err := parseXPath(check.expression)
		if err != nil {
			t.Fatalf("Parse(%q): %v", check.expression, err)
		}
		function, ok := expression.(*FunctionExpression)
		if !ok || function.Function.Name != check.name || len(function.Function.Arguments) != check.arguments {
			t.Fatalf("Parse(%q) = %#v, want %s with %d arguments", check.expression, expression, check.name, check.arguments)
		}
	}

	expression, err := parseXPath(`'01'='1'`)
	if err != nil {
		t.Fatal(err)
	}
	comparison, ok := expression.(*ComparisonExpression)
	if !ok {
		t.Fatalf("string equality parsed as %T, want *ComparisonExpression", expression)
	}
	if _, ok := comparison.Left.(*LiteralExpression); !ok {
		t.Fatalf("left operand parsed as %T, want string literal", comparison.Left)
	}
	if _, ok := comparison.Right.(*LiteralExpression); !ok {
		t.Fatalf("right operand parsed as %T, want string literal", comparison.Right)
	}
}

func TestUnifiedParserRejectsInvalidTypedConversionArity(t *testing.T) {
	for _, expression := range []string{
		`string('a', 'b')`,
		`number(1, 2)`,
		`boolean()`,
		`boolean(true(), false())`,
	} {
		if _, err := parseXPath(expression); err == nil {
			t.Errorf("Parse(%q) succeeded, want XPath function arity error", expression)
		}
	}
}

func TestXPathStringConversions(t *testing.T) {
	node := typedConversionContext()
	checks := []struct {
		expression string
		want       string
	}{
		{`string()`, " \t01\n"},
		{`string(*)`, "first"},
		{`string(@empty)`, ""},
		{`string(@missing)`, ""},
		{`string(true())`, "true"},
		{`string(number(' 001.0 '))`, "1"},
	}
	for _, check := range checks {
		if got := evaluateTypedExpression(t, node, check.expression); got != check.want {
			t.Errorf("%s = %q, want %q", check.expression, got, check.want)
		}
	}
}

func TestXPathNumberConversions(t *testing.T) {
	node := typedConversionContext()
	checks := []struct {
		expression string
		want       string
	}{
		{`number()`, "1"},
		{"number(' \t01\n')", "1"},
		{`number(@code)`, "1"},
		{`number(true())`, "1"},
		{`number(false())`, "0"},
		{`number('not-a-number')`, "NaN"},
		{`number(@missing)`, "NaN"},
	}
	for _, check := range checks {
		if got := evaluateTypedExpression(t, node, check.expression); got != check.want {
			t.Errorf("%s = %q, want %q", check.expression, got, check.want)
		}
	}
}

func TestXPathNumberUsesTheStrictXPathLexicalGrammar(t *testing.T) {
	node := typedConversionContext()
	for _, expression := range []string{
		`number('+1')`,
		`number('1e2')`,
		`number('Infinity')`,
		"number('\u00a01\u00a0')",
	} {
		if got := evaluateTypedExpression(t, node, expression); got != "NaN" {
			t.Errorf("%s = %q, want NaN", expression, got)
		}
	}
}

func TestXPathNumberSpecialValues(t *testing.T) {
	node := typedConversionContext()
	checks := []struct {
		expression string
		want       string
	}{
		{`string(number('-0'))`, "0"},
		{`boolean(number('-0'))`, "false"},
		{`string(1 div 0)`, "Infinity"},
		{`string((0 - 1) div 0)`, "-Infinity"},
		{`string(0 div 0)`, "NaN"},
		{`boolean(0 div 0)`, "false"},
	}
	for _, check := range checks {
		if got := evaluateTypedExpression(t, node, check.expression); got != check.want {
			t.Errorf("%s = %q, want %q", check.expression, got, check.want)
		}
	}
}

func TestXPathOverflowingNumericLiteralRoundsToInfinity(t *testing.T) {
	node := typedConversionContext()
	literal := strings.Repeat("9", 400)
	if got := evaluateTypedExpression(t, node, `string(`+literal+`)`); got != "Infinity" {
		t.Fatalf("overflowing numeric literal = %q, want Infinity", got)
	}
}

func TestXPathBooleanConversions(t *testing.T) {
	node := typedConversionContext()
	checks := []struct {
		expression string
		want       string
	}{
		{`boolean(@empty)`, "true"},
		{`boolean(@missing)`, "false"},
		{`boolean('')`, "false"},
		{`boolean('0')`, "true"},
		{`boolean(0)`, "false"},
		{`boolean(2)`, "true"},
		{`boolean(number('not-a-number'))`, "false"},
		{`boolean(string(false()))`, "true"},
	}
	for _, check := range checks {
		if got := evaluateTypedExpression(t, node, check.expression); got != check.want {
			t.Errorf("%s = %q, want %q", check.expression, got, check.want)
		}
	}
}

func TestXPathComparisonsRetainOperandTypes(t *testing.T) {
	node := typedConversionContext()
	checks := []struct {
		expression string
		want       string
	}{
		{`'01'='1'`, "false"},
		{`'01'!='1'`, "true"},
		{`number('01')=number('1')`, "true"},
		{`@code='1'`, "false"},
		{`@code=1`, "true"},
		{`@code=@plain`, "false"},
		{`@missing=false()`, "true"},
		{`@empty=false()`, "false"},
	}
	for _, check := range checks {
		if got := evaluateTypedExpression(t, node, check.expression); got != check.want {
			t.Errorf("%s = %q, want %q", check.expression, got, check.want)
		}
	}
}

func TestXPathRelationalComparisonsCoerceToNumbers(t *testing.T) {
	node := typedConversionContext()
	checks := []struct {
		expression string
		want       string
	}{
		{`'10'<'2'`, "false"},
		{`'10'>'2'`, "true"},
		{`true()<2`, "true"},
		{`@code<2`, "true"},
		{`number('bad')<2`, "false"},
	}
	for _, check := range checks {
		if got := evaluateTypedExpression(t, node, check.expression); got != check.want {
			t.Errorf("%s = %q, want %q", check.expression, got, check.want)
		}
	}
}

func TestXPathStringConversionSortsUnionNodeSetsInDocumentOrder(t *testing.T) {
	document := &types.Node{Type: types.DocumentNode, Name: "#document"}
	html := &types.Node{Type: types.ElementNode, Name: "html", Parent: document}
	body := &types.Node{Type: types.ElementNode, Name: "body", Parent: html}
	first := &types.Node{Type: types.ElementNode, Name: "z", TextContent: "first", Parent: body}
	second := &types.Node{Type: types.ElementNode, Name: "a", TextContent: "second", Parent: body}
	document.Children = []*types.Node{html}
	html.Children = []*types.Node{body}
	body.Children = []*types.Node{first, second}

	if got := evaluateTypedExpression(t, first, `string(//a | //z)`); got != "first" {
		t.Fatalf("string(//a | //z) = %q, want first node in document order", got)
	}
	if got := evaluateTypedExpression(t, first, `number(string((//a | //z)))`); got != "NaN" {
		t.Fatalf("nested union conversion = %q, want NaN from first node string", got)
	}
}

func TestXPathTypedAxesAndUnionsUseNodeSetDocumentOrderAndIdentity(t *testing.T) {
	document := &types.Node{Type: types.DocumentNode, Name: "#document"}
	outer := &types.Node{Type: types.ElementNode, Name: "div", TextContent: "AB", Parent: document}
	inner := &types.Node{Type: types.ElementNode, Name: "div", TextContent: "B", Parent: outer}
	span := &types.Node{Type: types.ElementNode, Name: "span", Parent: inner, Attributes: map[string]string{"x": "1"}, AttributeOrder: []string{"x"}}
	document.Children = []*types.Node{outer}
	outer.Children = []*types.Node{inner}
	inner.Children = []*types.Node{span}

	if got := evaluateTypedExpression(t, span, `string(ancestor::div)`); got != "AB" {
		t.Fatalf("string(ancestor::div) = %q, want outermost node's document-order value AB", got)
	}
	if got := evaluateTypedExpression(t, span, `string(ancestor::div[1])`); got != "B" {
		t.Fatalf("string(ancestor::div[1]) = %q, want nearest ancestor B", got)
	}
	if got := evaluateTypedExpression(t, span, `count(@x | @x)`); got != "1" {
		t.Fatalf("count(@x | @x) = %q, want one attribute identity", got)
	}
	if got := evaluateTypedExpression(t, span, `0 = 3 < 4`); got != "false" {
		t.Fatalf("0 = 3 < 4 = %q, want equality to bind below relational", got)
	}
}

func TestXPathEmptyNodeSetsAndTextNodeSetsRemainDistinctFromEmptyStrings(t *testing.T) {
	node := &types.Node{
		Type:       types.ElementNode,
		Name:       "div",
		Attributes: map[string]string{"empty": ""},
	}
	checks := []struct {
		expression string
		want       string
	}{
		{`@missing=''`, "false"},
		{`@missing=false()`, "true"},
		{`@empty=''`, "true"},
		{`@empty=false()`, "false"},
		{`text()=''`, "false"},
		{`string(text())=''`, "true"},
	}
	for _, check := range checks {
		if got := evaluateTypedExpression(t, node, check.expression); got != check.want {
			t.Errorf("%s = %q, want %q", check.expression, got, check.want)
		}
	}
}
