package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryProductionFormattingMismatchFamilies(t *testing.T) {
	testCases := []struct {
		name, document, expression string
		wantName, wantText         string
		wantStart, wantEnd         int
	}{
		{name: "b/span source", document: `<span><b>x</span>y</b>`, expression: `//span/b`, wantName: "b", wantText: "x", wantStart: 6, wantEnd: 10},
		{name: "b/span reconstructed", document: `<span><b>x</span>y</b>`, expression: `/html/body/b`, wantName: "b", wantText: "y", wantStart: 6, wantEnd: 22},
		{name: "a/li source", document: `<ul><li><a>x</li>y</a></ul>`, expression: `//li/a`, wantName: "a", wantText: "x", wantStart: 8, wantEnd: 12},
		{name: "a/li reconstructed", document: `<ul><li><a>x</li>y</a></ul>`, expression: `//ul/a`, wantName: "a", wantText: "y", wantStart: 8, wantEnd: 22},
		{name: "button/a source", document: `<a><button>x</a>y</button>`, expression: `/html/body/a`, wantName: "a", wantText: "", wantStart: 0, wantEnd: 16},
		{name: "button/a reconstructed", document: `<a><button>x</a>y</button>`, expression: `//button/a`, wantName: "a", wantText: "x", wantStart: 0, wantEnd: 0},
		{name: "span/small", document: `<small><span>x</small>y</span>`, expression: `//small/span`, wantName: "span", wantText: "x", wantStart: 7, wantEnd: 14},
		{name: "div/a source", document: `<a><div>x</a>y</div>`, expression: `/html/body/a`, wantName: "a", wantText: "", wantStart: 0, wantEnd: 13},
		{name: "div/a reconstructed", document: `<a><div>x</a>y</div>`, expression: `//div/a`, wantName: "a", wantText: "x", wantStart: 0, wantEnd: 0},
		{name: "font/b source", document: `<b><font>x</b>y</font>`, expression: `//b/font`, wantName: "font", wantText: "x", wantStart: 3, wantEnd: 10},
		{name: "font/b reconstructed", document: `<b><font>x</b>y</font>`, expression: `/html/body/font`, wantName: "font", wantText: "y", wantStart: 3, wantEnd: 22},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(testCase.expression, testCase.document)
			if err != nil || len(results) != 1 {
				t.Fatalf("Expected one recovered XPath result: results=%#v err=%v", results, err)
			}
			result := results[0]
			if result.NodeName != testCase.wantName || result.TextContent != testCase.wantText || result.StartLocation != testCase.wantStart || result.EndLocation != testCase.wantEnd {
				t.Fatalf("Expected <%s> text %q range %d:%d, got %#v", testCase.wantName, testCase.wantText, testCase.wantStart, testCase.wantEnd, result)
			}
		})
	}
}
