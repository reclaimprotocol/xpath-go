package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryNestedButtonStartProducesSiblingButtonsAndTail(t *testing.T) {
	const document = `<button id=a>x<button id=b>y</button>z`
	testCases := []struct {
		expression, name, text string
		start, end             int
	}{
		{expression: `/html/body/button[@id='a']`, name: "button", text: "x", start: 0, end: 14},
		{expression: `/html/body/button[@id='b']`, name: "button", text: "y", start: 14, end: 37},
		{expression: `/html/body/text()`, name: "#text", text: "z", start: 37, end: 38},
	}
	for _, testCase := range testCases {
		results, err := xpath.Query(testCase.expression, document)
		if err != nil || len(results) != 1 {
			t.Fatalf("Expected one result for %s: results=%#v err=%v", testCase.expression, results, err)
		}
		result := results[0]
		if result.NodeName != testCase.name || result.TextContent != testCase.text || result.StartLocation != testCase.start || result.EndLocation != testCase.end {
			t.Fatalf("Recovered result mismatch for %s: %#v", testCase.expression, result)
		}
	}
	results, err := xpath.Query(`/html/body/button/button`, document)
	if err != nil || len(results) != 0 {
		t.Fatalf("Second button must not remain nested: results=%#v err=%v", results, err)
	}
}

func TestQueryNestedButtonImpliedEndsAndFormattingReconstruction(t *testing.T) {
	testCases := []struct {
		name, document, expression string
		wantName, wantText         string
		wantStart, wantEnd         int
	}{
		{name: "implied paragraph end", document: `<button id=a><p>x<button id=b>y</button>z`, expression: `/html/body/button[@id='a']/p`, wantName: "p", wantText: "x", wantStart: 13, wantEnd: 17},
		{name: "button after implied ends", document: `<button id=a><p>x<button id=b>y</button>z`, expression: `/html/body/button[@id='b']`, wantName: "button", wantText: "y", wantStart: 17, wantEnd: 40},
		{name: "source formatting", document: `<button id=a><b>x<button id=b>y</button>z`, expression: `/html/body/button[@id='a']/b`, wantName: "b", wantText: "x", wantStart: 13, wantEnd: 17},
		{name: "reconstructed formatting", document: `<button id=a><b>x<button id=b>y</button>z`, expression: `/html/body/b`, wantName: "b", wantText: "yz", wantStart: 13, wantEnd: 41},
		{name: "button under reconstructed formatting", document: `<button id=a><b>x<button id=b>y</button>z`, expression: `/html/body/b/button[@id='b']`, wantName: "button", wantText: "y", wantStart: 17, wantEnd: 40},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(testCase.expression, testCase.document)
			if err != nil || len(results) != 1 {
				t.Fatalf("Expected one result: results=%#v err=%v", results, err)
			}
			result := results[0]
			if result.NodeName != testCase.wantName || result.TextContent != testCase.wantText || result.StartLocation != testCase.wantStart || result.EndLocation != testCase.wantEnd {
				t.Fatalf("Implied-end/formatting result mismatch: %#v", result)
			}
		})
	}
}

func TestQueryNestedButtonBoundariesRemainBrowserSafe(t *testing.T) {
	testCases := []struct {
		name, document, expression string
		wantCount                  int
	}{
		{name: "formatting", document: `<b><button id=a>x<button id=b>y</button>z</b>`, expression: `//b/button`, wantCount: 2},
		{name: "anchor", document: `<a><button id=a>x<button id=b>y</button>z</a>`, expression: `//a/button`, wantCount: 2},
		{name: "list", document: `<ul><li><button id=a>x<button id=b>y</button>z</li></ul>`, expression: `//li/button`, wantCount: 2},
		{name: "form", document: `<form><button id=a>x<button id=b>y</button>z</form>`, expression: `//form/button`, wantCount: 2},
		{name: "table", document: `<table><tr><td><button id=a>x<button id=b>y</button>z</td></tr></table>`, expression: `//td/button`, wantCount: 2},
		{name: "customizable select", document: `<select><button id=a>x<button id=b>y</button>z</select>`, expression: `//select/button`, wantCount: 2},
		{name: "select scope barrier", document: `<button id=a>x<select><button id=b>y</button></select>z</button>`, expression: `/html/body/button/select/button`, wantCount: 1},
		{name: "foreign integration", document: `<svg><foreignObject><button id=a>x<button id=b>y</button>z</foreignObject></svg>`, expression: `//*[local-name()='foreignObject']/button`, wantCount: 2},
		{name: "template isolation", document: `<template><button id=a>x<button id=b>y</button>z</template>`, expression: `//button`, wantCount: 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(testCase.expression, testCase.document)
			if err != nil || len(results) != testCase.wantCount {
				t.Fatalf("Boundary query mismatch: results=%#v err=%v", results, err)
			}
		})
	}
}
