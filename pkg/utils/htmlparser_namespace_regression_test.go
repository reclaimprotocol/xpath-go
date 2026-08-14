package utils

import "testing"

func TestParseSyntheticAndReconstructedHTMLNodesUseXHTMLNamespace(t *testing.T) {
	tableDoc, err := NewHTMLParser().Parse(`<table><tr><td>x</table>`)
	if err != nil {
		t.Fatal(err)
	}
	tbody := formattingElements(tableDoc, "tbody")[0]
	if tbody.StartPos != 0 || tbody.EndPos != 0 {
		t.Fatalf("Expected synthetic tbody location to remain empty: %#v", tbody)
	}
	requireNodeNamespace(t, tbody, htmlNamespaceURI)

	formattingDoc, err := NewHTMLParser().Parse(`<p><b>x<div>y</div>`)
	if err != nil {
		t.Fatal(err)
	}
	bold := formattingElements(formattingDoc, "b")
	if len(bold) < 2 || bold[1].Parent == nil || bold[1].Parent.Name != "div" {
		t.Fatalf("Expected a reconstructed b under div: %#v", bold)
	}
	requireNodeNamespace(t, bold[1], htmlNamespaceURI)
}
