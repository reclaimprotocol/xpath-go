package xpath_test

import (
	"reflect"
	"strings"
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func batch26XPathOptions(t *testing.T, scriptingEnabled bool) xpath.Options {
	t.Helper()
	opts := xpath.Options{IncludeLocation: true, OutputFormat: "nodes"}
	field := reflect.ValueOf(&opts).Elem().FieldByName("ScriptingEnabled")
	if !field.IsValid() {
		t.Fatal("xpath.Options must expose ScriptingEnabled bool")
	}
	if field.Kind() != reflect.Bool || !field.CanSet() {
		t.Fatalf("Options.ScriptingEnabled must be a settable bool, got %s", field.Type())
	}
	field.SetBool(scriptingEnabled)
	return opts
}

func TestQueryNoscriptDefaultsToJSDOMScriptingDisabledMode(t *testing.T) {
	const content = `<html><head></head><body><noscript id=n><b id=b>x&amp;y</b></noscript><p id=p>z</p></body></html>`
	results, err := xpath.Query(`/html/body/noscript/b[@id='b'] | /html/body/p[@id='p']`, content)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].NodeName != "b" || results[0].TextContent != "x&y" || results[1].NodeName != "p" || results[1].TextContent != "z" {
		t.Fatalf("default disabled noscript XPath mismatch: %#v", results)
	}
}

func TestQueryNoscriptCanSelectChromeScriptingEnabledMode(t *testing.T) {
	const content = `<html><head></head><body><noscript id=n><b id=b>x&amp;y</b></noscript><p id=p>z</p></body></html>`
	results, err := xpath.QueryWithOptions(`/html/body/noscript/text() | //*[@id='b'] | //*[@id='p']`, content, batch26XPathOptions(t, true))
	if err != nil {
		t.Fatal(err)
	}
	rawStart := strings.Index(content, `<b id=b>`)
	rawEnd := strings.Index(content, `</noscript>`)
	if len(results) != 2 || results[0].NodeType != 3 || results[0].TextContent != content[rawStart:rawEnd] || results[0].StartLocation != rawStart || results[0].EndLocation != rawEnd || results[1].NodeName != "p" {
		t.Fatalf("enabled noscript XPath/raw location mismatch: %#v", results)
	}
}

func TestQueryHeadNoscriptModesExposeDifferentTrees(t *testing.T) {
	const content = `<html><head><noscript id=n><meta id=m></noscript></head><body><p id=p>z</p></body></html>`
	disabled, err := xpath.Query(`/html/head/noscript/meta[@id='m'] | /html/body/p`, content)
	if err != nil {
		t.Fatal(err)
	}
	if len(disabled) != 2 || disabled[0].NodeName != "meta" || disabled[1].NodeName != "p" {
		t.Fatalf("disabled head-noscript XPath tree mismatch: %#v", disabled)
	}

	enabled, err := xpath.QueryWithOptions(`/html/head/noscript/text() | //*[@id='m'] | /html/body/p`, content, batch26XPathOptions(t, true))
	if err != nil {
		t.Fatal(err)
	}
	if len(enabled) != 2 || enabled[0].NodeType != 3 || enabled[0].TextContent != `<meta id=m>` || enabled[1].NodeName != "p" {
		t.Fatalf("enabled head-noscript XPath tree mismatch: %#v", enabled)
	}
}
