package xpath_test

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go"
)

func requireProductionRecoveryQuery(t *testing.T, document, expression, name, text string, start, end int) xpath.Result {
	t.Helper()
	results, err := xpath.Query(expression, document)
	if err != nil {
		t.Fatalf("Query %s failed: %v", expression, err)
	}
	if len(results) != 1 {
		t.Fatalf("Expected one result for %s, got %#v", expression, results)
	}
	result := results[0]
	if result.NodeName != name || result.TextContent != text || result.StartLocation != start || result.EndLocation != end {
		t.Fatalf("Expected %s text %q at %d:%d for %s, got %#v", name, text, start, end, expression, result)
	}
	return result
}

func TestQueryProductionRecoveryClosesBlockThroughFooterAndAnchor(t *testing.T) {
	const document = `<html><body><div><footer><a>x</div>y</a></footer></body></html>`
	requireProductionRecoveryQuery(t, document, `//body/div`, "div", "x", 12, 35)
	requireProductionRecoveryQuery(t, document, `//body/div/footer`, "footer", "x", 17, 29)
	requireProductionRecoveryQuery(t, document, `//body/div/footer/a`, "a", "x", 25, 29)
	requireProductionRecoveryQuery(t, document, `//body/div/following-sibling::a`, "a", "y", 25, 40)
	requireProductionRecoveryQuery(t, document, `//body/div/following-sibling::a/text()`, "#text", "y", 35, 36)
}

func TestQueryProductionRecoveryClosesListThroughNestedNav(t *testing.T) {
	const document = `<ul><li>x<nav><li>y</ul>z</nav>`
	requireProductionRecoveryQuery(t, document, `//ul`, "ul", "xy", 0, 24)
	requireProductionRecoveryQuery(t, document, `//ul/li/nav`, "nav", "y", 9, 19)
	requireProductionRecoveryQuery(t, document, `//ul/li/nav/li`, "li", "y", 14, 19)
	requireProductionRecoveryQuery(t, document, `//ul/following-sibling::text()`, "#text", "z", 24, 25)
}

func TestQueryProductionRecoveryKeepsLessThanInTagName(t *testing.T) {
	const document = `<div><span<em>x</span>y</div>`
	requireProductionRecoveryQuery(t, document, `//div/*`, "span<em", "xy", 5, 23)
	requireProductionRecoveryQuery(t, document, `//div/*/text()`, "#text", "xy", 14, 23)
}

func TestQueryProductionRecoveryHandlesGenericEndTagsLikeBrowser(t *testing.T) {
	const matching = `<div><span></div>`
	requireProductionRecoveryQuery(t, matching, `//div`, "div", "", 0, 17)
	requireProductionRecoveryQuery(t, matching, `//div/span`, "span", "", 5, 11)

	const absent = `<div></span></div>`
	requireProductionRecoveryQuery(t, absent, `//div`, "div", "", 0, 18)
	results, err := xpath.Query(`//span`, absent)
	if err != nil || len(results) != 0 {
		t.Fatalf("Absent generic end must not create a node: results=%#v err=%v", results, err)
	}
}

func TestQueryProductionRecoveryReconstructsFormattingAfterGenericPop(t *testing.T) {
	const document = `<b><span><small>x</span>y</b>`
	requireProductionRecoveryQuery(t, document, `//b`, "b", "xy", 0, 29)
	requireProductionRecoveryQuery(t, document, `//b/span/small`, "small", "x", 9, 17)
	requireProductionRecoveryQuery(t, document, `//b/small`, "small", "y", 9, 25)
	requireProductionRecoveryQuery(t, document, `//b/small/text()`, "#text", "y", 24, 25)
}

func TestQueryProductionRecoveryPreservesFormPointerAcrossBlockUnwind(t *testing.T) {
	const document = `<div><form id=a>x</div><form id=b>y</form><form id=c>z</form>`
	requireProductionRecoveryQuery(t, document, `//div/form[@id='a']`, "form", "x", 5, 17)
	requireProductionRecoveryQuery(t, document, `//div/following-sibling::text()`, "#text", "y", 34, 35)
	requireProductionRecoveryQuery(t, document, `//form[@id='c']`, "form", "z", 42, 61)
	results, err := xpath.Query(`//form[@id='b']`, document)
	if err != nil || len(results) != 0 {
		t.Fatalf("Form b start must be ignored while stale pointer is set: results=%#v err=%v", results, err)
	}
}

func TestQueryProductionRecoverySwallowsMissingGreaterThanIntoAttributes(t *testing.T) {
	unquoted := requireProductionRecoveryQuery(t, `<div class=x<span>y</span></div>`, `//div`, "div", "y", 0, 32)
	if unquoted.Attributes["class"] != "x<span" || len(unquoted.Attributes) != 1 {
		t.Fatalf("Unquoted missing-> attributes mismatch: %#v", unquoted.Attributes)
	}

	quoted := requireProductionRecoveryQuery(t, `<div class="x"<span>y</span></div>`, `//div`, "div", "y", 0, 34)
	if quoted.Attributes["class"] != "x" || quoted.Attributes["<span"] != "" || len(quoted.Attributes) != 2 {
		t.Fatalf("Quoted missing-> attributes mismatch: %#v", quoted.Attributes)
	}
	results, err := xpath.Query(`//span`, `<div class="x"<span>y</span></div>`)
	if err != nil || len(results) != 0 {
		t.Fatalf("Swallowed span token emitted an element: results=%#v err=%v", results, err)
	}
}

func TestQueryProductionRecoveryIgnoredEndAndDocumentEOF(t *testing.T) {
	const recovered = `<div></span></div><foo>x`
	requireProductionRecoveryQuery(t, recovered, `/html/body/foo`, "foo", "x", 18, len(recovered))
	requireProductionRecoveryQuery(t, `<div><span></div>`, `//div/span`, "span", "", 5, 11)
}
