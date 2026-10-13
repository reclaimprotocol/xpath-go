package xpath

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestScalarExpressionsAcrossPublicAPIs(t *testing.T) {
	data, err := os.ReadFile("tests/shared/scalar_testcases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name, HTML, XPath, Want string
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			compiled, err := Compile(tc.XPath)
			if err != nil {
				t.Fatal(err)
			}
			check := func(results []Result, err error) {
				t.Helper()
				want := []Result{{Value: tc.Want, TextContent: tc.Want}}
				if err != nil || !reflect.DeepEqual(results, want) {
					t.Fatalf("%s = %#v, %v; want %#v", tc.XPath, results, err, want)
				}
			}
			check(Query(tc.XPath, tc.HTML))
			check(QueryBytes(tc.XPath, []byte(tc.HTML)))
			check(compiled.Evaluate(tc.HTML))
			for _, format := range []string{"", "nodes", "values", "paths"} {
				for _, includeLocation := range []bool{false, true} {
					for _, contentsOnly := range []bool{false, true} {
						opts := Options{OutputFormat: format, IncludeLocation: includeLocation, ContentsOnly: contentsOnly}
						check(QueryWithOptions(tc.XPath, tc.HTML, opts))
						check(QueryBytesWithOptions(tc.XPath, []byte(tc.HTML), opts))
						check(compiled.EvaluateWithOptions(tc.HTML, opts))
						check(compiled.EvaluateBytesWithOptions([]byte(tc.HTML), opts))
					}
				}
			}
		})
	}
}

func TestScalarCompiledReuseWithCharset(t *testing.T) {
	compiled, err := Compile(`concat(//input/@value, '!')`)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Andr\xe9", "Zo\xeb"} {
		name := name
		t.Run(fmt.Sprintf("name-%x", name), func(t *testing.T) {
			t.Parallel()
			content := []byte(`<input value="` + name + `">`)
			for iteration := 0; iteration < 10; iteration++ {
				got, err := compiled.EvaluateBytesWithOptions(content, Options{Charset: "windows-1252", IncludeLocation: true})
				want := "André!"
				if name == "Zo\xeb" {
					want = "Zoë!"
				}
				if err != nil || !reflect.DeepEqual(got, []Result{{Value: want, TextContent: want}}) {
					t.Fatalf("got %#v, %v; want %q", got, err, want)
				}
			}
		})
	}
}

func TestScalarSupportPreservesNodeValidation(t *testing.T) {
	for _, expr := range []string{
		`//p | 'text'`, `1 | //p`, `('text')[1]`, `concat('a', 'b')/text()`,
		`string(//p | 'text')`, `//p[('text')[1]]`,
		`substring-before()`, `substring-before('a')`, `substring-before('a', 'b', 'c')`,
		`substring-after()`, `substring-after('a')`, `substring-after('a', 'b', 'c')`,
	} {
		if _, err := Compile(expr); err == nil {
			t.Errorf("Compile(%q) accepted an invalid expression", expr)
		}
	}
	results, err := Query(`//missing`, `<p>text</p>`)
	if err != nil || len(results) != 0 {
		t.Fatalf("empty node selection = %#v, %v", results, err)
	}
	results, err = Query(`//p[substring-before(@data, ':')='keep' and substring-after(@data, ':')='yes']`, `<p data="skip:yes">no</p><p data="keep:yes">match</p>`)
	if err != nil || len(results) != 1 || results[0].TextContent != "match" || results[0].NodeType != 1 {
		t.Fatalf("substring predicate selection = %#v, %v", results, err)
	}
}

func TestScalarDocumentContext(t *testing.T) {
	for _, expression := range []string{
		`position()`, `last()`,
		`count(//p[position()=1]) + position() - last()`,
	} {
		results, err := Query(expression, `<p>first</p><p>second</p>`)
		if err != nil || !reflect.DeepEqual(results, []Result{{Value: "1", TextContent: "1"}}) {
			t.Fatalf("%s = %#v, %v; want 1", expression, results, err)
		}
	}
}
