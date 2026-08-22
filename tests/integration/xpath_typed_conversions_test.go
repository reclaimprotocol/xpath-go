package xpath_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	xpath "github.com/reclaimprotocol/xpath-go"
)

type typedConversionCase struct {
	Name    string   `json:"name"`
	HTML    string   `json:"html"`
	XPath   string   `json:"xpath"`
	WantIDs []string `json:"want_ids"`
}

func loadTypedConversionCases(t *testing.T) []typedConversionCase {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate typed conversion test source")
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(filename), "..", "shared", "typed_conversion_testcases.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []typedConversionCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	return cases
}

func resultIDs(results []xpath.Result) []string {
	ids := make([]string, len(results))
	for i, result := range results {
		ids[i] = result.Attributes["id"]
	}
	return ids
}

func TestQueryTypedConversionsMatchesSharedBrowserCases(t *testing.T) {
	for _, testCase := range loadTypedConversionCases(t) {
		t.Run(testCase.Name, func(t *testing.T) {
			results, err := xpath.Query(testCase.XPath, testCase.HTML)
			if err != nil {
				t.Fatalf("Query(%q): %v", testCase.XPath, err)
			}
			got := resultIDs(results)
			if fmt.Sprint(got) != fmt.Sprint(testCase.WantIDs) {
				t.Fatalf("Query(%q) IDs = %v, want browser-backed %v", testCase.XPath, got, testCase.WantIDs)
			}
		})
	}
}

func TestQueryChromeXPathTypedEdgeSemantics(t *testing.T) {
	tests := []struct {
		name     string
		document string
		expr     string
		wantIDs  []string
	}{
		{
			name:     "empty node-set is not an empty string",
			document: `<main><div id='missing'></div><div id='empty' value=''></div></main>`,
			expr:     `//div[@missing='' or @value='']`,
			wantIDs:  []string{"empty"},
		},
		{
			name:     "empty node-set equals false after boolean precedence",
			document: `<main><div id='missing'></div><div id='empty' value=''></div></main>`,
			expr:     `//div[@missing=false()]`,
			wantIDs:  []string{"missing", "empty"},
		},
		{
			name:     "absent text node differs from its string conversion",
			document: `<main><div id='none'></div><div id='empty-string'></div></main>`,
			expr:     `//div[not(text()='') and string(text())='']`,
			wantIDs:  []string{"none", "empty-string"},
		},
		{
			name:     "strict number grammar rejects non-XPath spellings",
			document: `<main><div id='match'></div></main>`,
			expr:     "//div[number('+1')!=number('+1') and number('1e2')!=number('1e2') and number('Infinity')!=number('Infinity') and number('\u00a01\u00a0')!=number('\u00a01\u00a0')]",
			wantIDs:  []string{"match"},
		},
		{
			name:     "NaN negative zero and infinities keep numeric semantics",
			document: `<main><div id='match'></div></main>`,
			expr:     `//div[not(boolean(number('-0'))) and not(boolean(0 div 0)) and 1 div 0 > 0 and (0 - 1) div 0 < 0]`,
			wantIDs:  []string{"match"},
		},
		{
			name:     "existential node-set equality and inequality can both hold",
			document: `<main><case id='match'><left>a</left><left>b</left><right>a</right><right>c</right></case></main>`,
			expr:     `//case[left=right and left!=right]`,
			wantIDs:  []string{"match"},
		},
		{
			name:     "boolean comparison takes precedence over strings",
			document: `<main><div id='match'></div></main>`,
			expr:     `//div['false'=true() and @missing=false()]`,
			wantIDs:  []string{"match"},
		},
		{
			name:     "numeric predicate result means context position",
			document: `<ol><li id='first' pick='2'></li><li id='second' pick='2'></li><li id='third' pick='2'></li></ol>`,
			expr:     `//li[number(@pick)]`,
			wantIDs:  []string{"second"},
		},
		{
			name:     "union string conversion uses document order",
			document: `<main id='match'><z>first</z><a>second</a></main>`,
			expr:     `//main[string(//a | //z)='first']`,
			wantIDs:  []string{"match"},
		},
		{
			name:     "nested conversions consume a union node-set",
			document: `<main id='match'><z><b> 01 </b></z><a>2</a></main>`,
			expr:     `//main[number(string((//a | //b)))=1]`,
			wantIDs:  []string{"match"},
		},
		{
			name:     "relational node-set comparison is existential and numeric",
			document: `<main><case id='match'><left>10</left><left>1</left><right>2</right></case></main>`,
			expr:     `//case[left<right]`,
			wantIDs:  []string{"match"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			results, err := xpath.Query(testCase.expr, testCase.document)
			if err != nil {
				t.Fatal(err)
			}
			if got := resultIDs(results); fmt.Sprint(got) != fmt.Sprint(testCase.wantIDs) {
				t.Fatalf("IDs = %v, want %v", got, testCase.wantIDs)
			}
		})
	}
}

func TestQueryRejectsInvalidTypedConversionArity(t *testing.T) {
	for _, expression := range []string{
		`//div[string('a', 'b')]`,
		`//div[number(1, 2)]`,
		`//div[boolean()]`,
		`//div[boolean(true(), false())]`,
	} {
		if _, err := xpath.Query(expression, `<div></div>`); err == nil {
			t.Errorf("Query(%q) succeeded, want XPath function arity error", expression)
		}
	}
}

func TestCompiledTypedConversionQueryCanBeReused(t *testing.T) {
	compiled, err := xpath.Compile(`//item[number(@rank)=1 and boolean(@enabled) and string(@code)='01']`)
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		document string
		wantID   string
	}{
		{`<item id='first' rank=' 1 ' enabled='' code='01'></item><item id='skip' rank='2' enabled='' code='01'></item>`, "first"},
		{`<item id='skip' rank='1' code='01'></item><item id='second' rank='01' enabled='' code='01'></item>`, "second"},
	}
	for i, check := range checks {
		results, err := compiled.Evaluate(check.document)
		if err != nil {
			t.Fatalf("reuse %d: %v", i, err)
		}
		if got := resultIDs(results); len(got) != 1 || got[0] != check.wantID {
			t.Fatalf("reuse %d IDs = %v, want [%s]", i, got, check.wantID)
		}
	}
}

func typedConversionDocument(elements int) string {
	var document strings.Builder
	document.Grow(elements * 52)
	for i := 0; i < elements; i++ {
		fmt.Fprintf(&document, `<item id='n%d' value=' %d ' enabled=''></item>`, i, i)
	}
	return document.String()
}

func bestTypedConversionDuration(t *testing.T, document string, want int) time.Duration {
	t.Helper()
	best := time.Duration(1<<63 - 1)
	for i := 0; i < 3; i++ {
		start := time.Now()
		results, err := xpath.Query(`//item[number(@value)>=0 and boolean(@enabled)]`, document)
		elapsed := time.Since(start)
		if err != nil || len(results) != want {
			t.Fatalf("typed conversion scaling query returned %d nodes, want %d; err=%v", len(results), want, err)
		}
		if elapsed < best {
			best = elapsed
		}
	}
	return best
}

func TestQueryTypedConversionScaling(t *testing.T) {
	const smallN, largeN = 500, 2000
	smallDocument := typedConversionDocument(smallN)
	largeDocument := typedConversionDocument(largeN)
	_ = bestTypedConversionDuration(t, smallDocument, smallN)
	small := bestTypedConversionDuration(t, smallDocument, smallN)
	large := bestTypedConversionDuration(t, largeDocument, largeN)
	t.Logf("typed conversion scaling: 500=%v 2000=%v", small, large)
	if large > 10*small+50*time.Millisecond {
		t.Fatalf("typed conversion predicates scale superlinearly: 500=%v 2000=%v", small, large)
	}
}
