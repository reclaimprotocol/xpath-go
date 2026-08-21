package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryRootNoscriptRoutesToHeadInBothModes(t *testing.T) {
	const content = `<noscript id=n><meta id=m></noscript><p id=p>x</p>`
	disabled, err := xpath.Query(`/html/head/noscript/meta[@id='m'] | /html/body/p`, content)
	if err != nil {
		t.Fatal(err)
	}
	if len(disabled) != 2 || disabled[0].NodeName != "meta" || disabled[1].NodeName != "p" {
		t.Fatalf("default root noscript routing mismatch: %#v", disabled)
	}

	enabled, err := xpath.QueryWithOptions(`/html/head/noscript/text() | //*[@id='m'] | /html/body/p`, content, batch26XPathOptions(t, true))
	if err != nil {
		t.Fatal(err)
	}
	if len(enabled) != 2 || enabled[0].NodeType != 3 || enabled[0].TextContent != `<meta id=m>` || enabled[1].NodeName != "p" {
		t.Fatalf("enabled root noscript routing mismatch: %#v", enabled)
	}
}

func TestQueryWithZeroValueOptionsKeepsScriptingDisabled(t *testing.T) {
	const content = `<html><head></head><body><noscript id=n><b id=b>x</b></noscript></body></html>`
	results, err := xpath.QueryWithOptions(`/html/body/noscript/b`, content, xpath.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Attributes["id"] != "b" || results[0].TextContent != "x" {
		t.Fatalf("zero-value Options must keep scripting disabled: %#v", results)
	}
}

func TestQueryScriptingOptionDoesNotLeakAcrossCalls(t *testing.T) {
	const content = `<html><head></head><body><noscript id=n><b id=b>x</b></noscript></body></html>`
	enabled, err := xpath.QueryWithOptions(`/html/body/noscript/text() | //*[@id='b']`, content, batch26XPathOptions(t, true))
	if err != nil || len(enabled) != 1 || enabled[0].NodeType != 3 || enabled[0].TextContent != `<b id=b>x</b>` {
		t.Fatalf("enabled option result mismatch: %#v err=%v", enabled, err)
	}
	disabled, err := xpath.Query(`/html/body/noscript/b`, content)
	if err != nil || len(disabled) != 1 || disabled[0].Attributes["id"] != "b" {
		t.Fatalf("enabled option leaked into default Query: %#v err=%v", disabled, err)
	}
}
