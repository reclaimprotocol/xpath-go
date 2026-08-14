package xpath_test

import (
	"strings"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

var extendedRawTextQueryTags = []string{"xmp", "iframe", "noembed", "noframes"}

func assertExtendedTextQuery(t *testing.T, document, expression, wantValue string, wantStart, wantEnd int) {
	t.Helper()
	results, err := xpath.Query(expression, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].NodeName != "#text" || results[0].TextContent != wantValue {
		t.Fatalf("Expected one text value %q, got %#v", wantValue, results)
	}
	if results[0].StartLocation != wantStart || results[0].EndLocation != wantEnd {
		t.Fatalf("Expected original UTF-8 range %d:%d, got %#v", wantStart, wantEnd, results[0])
	}
}

func TestQueryRemainingRawTextElementsKeepMarkupAndReferencesLiteral(t *testing.T) {
	const raw = `é<b>&amp;&#65;`
	for _, tag := range extendedRawTextQueryTags {
		t.Run(tag, func(t *testing.T) {
			opening := `<` + tag + `>`
			document := opening + raw + `</` + strings.ToUpper(tag) + `><p>after</p>`
			assertExtendedTextQuery(t, document, `//`+tag+`/text()`, raw, len(opening), len(opening)+len(raw))
			paragraphs, err := xpath.Query(`//p/text()`, document)
			if err != nil || len(paragraphs) != 1 || paragraphs[0].TextContent != "after" {
				t.Fatalf("Expected parsing to continue with p after <%s>, results=%#v err=%v", tag, paragraphs, err)
			}
		})
	}
}

func TestQueryRemainingRawTextElementsKeepNonmatchingEndTagsLiteral(t *testing.T) {
	for _, tag := range extendedRawTextQueryTags {
		t.Run(tag, func(t *testing.T) {
			opening := `<` + tag + `>`
			raw := `x</` + tag + `-x>y`
			document := opening + raw + `</` + tag + `>`
			assertExtendedTextQuery(t, document, `//`+tag+`/text()`, raw, len(opening), len(opening)+len(raw))
		})
	}
}

func TestQueryRemainingRawTextElementsReplaceNUL(t *testing.T) {
	for _, tag := range extendedRawTextQueryTags {
		t.Run(tag, func(t *testing.T) {
			opening := `<` + tag + `>`
			document := opening + "a\x00b</" + tag + `>`
			assertExtendedTextQuery(t, document, `//`+tag+`/text()`, "a\uFFFDb", len(opening), len(opening)+3)
		})
	}
}

func TestQueryRemainingRawTextElementsCloseAtEOF(t *testing.T) {
	const raw = `x&amp;<b>`
	for _, tag := range extendedRawTextQueryTags {
		t.Run(tag, func(t *testing.T) {
			opening := `<` + tag + `>`
			document := opening + raw
			assertExtendedTextQuery(t, document, `//`+tag+`/text()`, raw, len(opening), len(document))
		})
	}
}

func TestQueryPlaintextConsumesEverythingThroughEOF(t *testing.T) {
	const opening = `<plaintext>`
	const raw = "é\n<b>x</b>&amp;&#65;</plaintext><p>after</p>"
	document := opening + raw
	assertExtendedTextQuery(t, document, `//plaintext/text()`, raw, len(opening), len(document))
	paragraphs, err := xpath.Query(`//p`, document)
	if err != nil {
		t.Fatalf("Query for apparent following p returned an error: %v", err)
	}
	if len(paragraphs) != 0 {
		t.Fatalf("Expected apparent p source to remain plaintext text, got %#v", paragraphs)
	}
}

func TestQueryPlaintextReplacesNULThroughEOF(t *testing.T) {
	const document = "<plaintext>a\x00b"
	assertExtendedTextQuery(t, document, `//plaintext/text()`, "a\uFFFDb", len(`<plaintext>`), len(document))
}

func TestQueryNestedExtendedTextStatesCloseAllAncestorsAtEOF(t *testing.T) {
	tags := append(append([]string{}, extendedRawTextQueryTags...), "plaintext")
	for _, tag := range tags {
		t.Run(tag, func(t *testing.T) {
			document := `<div><` + tag + `>x`
			innerStart := len(`<div>`)
			textStart := innerStart + len(`<`+tag+`>`)
			assertExtendedTextQuery(t, document, `//`+tag+`/text()`, "x", textStart, len(document))

			divs, err := xpath.Query(`//div`, document)
			if err != nil || len(divs) != 1 || divs[0].TextContent != "x" || divs[0].StartLocation != 0 || divs[0].EndLocation != len(document) {
				t.Fatalf("Expected outer div x at 0:%d, results=%#v err=%v", len(document), divs, err)
			}
			inners, err := xpath.Query(`//`+tag, document)
			if err != nil || len(inners) != 1 || inners[0].TextContent != "x" || inners[0].StartLocation != innerStart || inners[0].EndLocation != len(document) {
				t.Fatalf("Expected inner <%s> x at %d:%d, results=%#v err=%v", tag, innerStart, len(document), inners, err)
			}
		})
	}
}

func TestQueryEmptyExtendedTextStatesLikeBrowser(t *testing.T) {
	testCases := []struct {
		tag, document string
	}{
		{tag: "xmp", document: `<xmp></xmp>`},
		{tag: "plaintext", document: `<plaintext>`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.tag, func(t *testing.T) {
			elements, err := xpath.Query(`//`+testCase.tag, testCase.document)
			if err != nil || len(elements) != 1 || elements[0].TextContent != "" || elements[0].StartLocation != 0 || elements[0].EndLocation != len(testCase.document) {
				t.Fatalf("Expected empty <%s> at 0:%d, results=%#v err=%v", testCase.tag, len(testCase.document), elements, err)
			}
			texts, err := xpath.Query(`//`+testCase.tag+`/text()`, testCase.document)
			if err != nil || len(texts) != 0 {
				t.Fatalf("Expected no text child for empty <%s>, results=%#v err=%v", testCase.tag, texts, err)
			}
		})
	}
}

func TestQueryPlaintextCRLFAndSupplementarySourceLocation(t *testing.T) {
	const document = "<plaintext>🙂\r\nz"
	assertExtendedTextQuery(t, document, `//plaintext/text()`, "🙂\nz", len(`<plaintext>`), len(document))
}

func TestQueryExtendedRawTextEndTagDelimiterAndIncompleteEOF(t *testing.T) {
	t.Run("whitespace delimiter", func(t *testing.T) {
		const document = `<iframe>x</IFRAME foo=bar><p>y</p>`
		assertExtendedTextQuery(t, document, `//iframe/text()`, "x", len(`<iframe>`), len(`<iframe>x`))
		paragraphs, err := xpath.Query(`//p/text()`, document)
		if err != nil || len(paragraphs) != 1 || paragraphs[0].TextContent != "y" {
			t.Fatalf("Expected parsing to continue with p, results=%#v err=%v", paragraphs, err)
		}
	})

	t.Run("incomplete EOF candidate", func(t *testing.T) {
		const document = `<iframe>x</iframe `
		assertExtendedTextQuery(t, document, `//iframe/text()`, "x", len(`<iframe>`), len(document))
	})
}
