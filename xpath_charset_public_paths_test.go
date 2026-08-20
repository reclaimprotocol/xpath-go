package xpath_test

import (
	"fmt"
	"sync"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

// These tests pin the two public paths that need to share the charset-aware
// parser: QueryWithOptions ContentsOnly and compiled EvaluateWithOptions.
// The latter is also the API shape documented by docs/CONTENTS_ONLY.md.
func TestQueryISO88591ContentsOnlyUsesDecodedTextAndRawBoundaries(t *testing.T) {
	document := []byte(`<main><div title="Se`)
	document = append(document, 0xF1)
	document = append(document, []byte(`or">Jos`)...)
	document = append(document, 0xE9)
	document = append(document, []byte(`</div></main>`)...)

	results, err := xpath.QueryWithOptions(
		`//div[@title='Señor' and text()='José']`,
		string(document),
		xpath.Options{IncludeLocation: true, OutputFormat: "nodes", ContentsOnly: true, Charset: "iso-8859-1"},
	)
	if err != nil {
		t.Fatalf("charset-aware contents-only query failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one decoded div, got %#v", results)
	}
	// These are offsets into the raw ISO-8859-1 byte stream. Do not use len
	// on the Unicode literals in the XPath or expected value: ñ/é occupy one
	// source byte each, but two UTF-8 bytes in a Go string literal.
	wantStart := len(`<main><div title="Se`) + 1 + len(`or">`)
	wantEnd := wantStart + len(`Jos`) + 1
	got := results[0]
	if got.Value != "José" || got.TextContent != "José" || got.StartLocation != wantStart || got.EndLocation != wantEnd {
		t.Fatalf("contents-only result must expose decoded text and raw boundaries: %#v (want %d:%d)", got, wantStart, wantEnd)
	}
}

func TestCompiledISO88591EvaluateWithOptionsUsesCharset(t *testing.T) {
	document := []byte(`<div title="Se`)
	document = append(document, 0xF1)
	document = append(document, []byte(`or">Jos`)...)
	document = append(document, 0xE9)
	document = append(document, []byte(`</div>`)...)

	compiled, err := xpath.Compile(`//div[@title='Señor']/text()`)
	if err != nil {
		t.Fatal(err)
	}
	results, err := compiled.EvaluateWithOptions(string(document), xpath.Options{
		IncludeLocation: true,
		OutputFormat:    "nodes",
		Charset:         "iso-8859-1",
	})
	if err != nil {
		t.Fatalf("compiled charset-aware evaluation failed: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != "José" {
		t.Fatalf("compiled evaluation must use decoded text/attrs: %#v", results)
	}
}

func TestCompiledEvaluateWithOptionsIsSafeForConcurrentDocuments(t *testing.T) {
	compiled, err := xpath.Compile(`//p/text()`)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		raw     []byte
		charset string
		want    string
	}{
		{raw: []byte(`<p>plain</p>`), want: "plain"},
		{raw: []byte{'<', 'p', '>', 'J', 'o', 's', 0xe9, '<', '/', 'p', '>'}, charset: "iso-8859-1", want: "José"},
	}

	const workers = 16
	const iterations = 100
	errs := make(chan error, workers*iterations)
	var wg sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for iteration := 0; iteration < iterations; iteration++ {
				check := cases[(worker+iteration)%len(cases)]
				results, evalErr := compiled.EvaluateBytesWithOptions(check.raw, xpath.Options{
					IncludeLocation: true,
					OutputFormat:    "nodes",
					Charset:         check.charset,
				})
				if evalErr != nil {
					errs <- evalErr
					continue
				}
				if len(results) != 1 || results[0].TextContent != check.want {
					errs <- fmt.Errorf("worker %d iteration %d: results=%#v want %q", worker, iteration, results, check.want)
				}
			}
		}(worker)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
