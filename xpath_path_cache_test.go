package xpath

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestConvertNodesToResultsDoesNotAliasOriginlessValues(t *testing.T) {
	root := &types.Node{Type: types.DocumentNode, Name: "#document"}
	// The conversion boundary accepts values, so these deliberately do not
	// retain Origin pointers. This mirrors callers that construct Nodes in
	// tests or adapters rather than through the evaluator.
	nodes := []types.Node{
		{Type: types.ElementNode, Name: "first", Parent: root},
		{Type: types.ElementNode, Name: "second", Parent: root},
	}

	results := convertNodesToResults(nodes, Options{OutputFormat: "paths"}, "")
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0].Path != "/#document/first" || results[1].Path != "/#document/second" {
		t.Fatalf("origin-less paths = %#v, want distinct paths", results)
	}
	if results[0].Value == results[1].Value {
		t.Fatalf("origin-less result paths aliased: %#v", results)
	}
}

func TestResultPathsUseOriginalSiblingIdentity(t *testing.T) {
	results, err := Query(`//div`, `<main><div>first</div><div>second</div></main>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	want := []string{
		`/#document/html/body/main/div[1]`,
		`/#document/html/body/main/div[2]`,
	}
	for index := range want {
		if results[index].Path != want[index] {
			t.Errorf("result %d path = %q, want %q", index, results[index].Path, want[index])
		}
	}
}

func TestAttributeResultsRetainIdentityAndPath(t *testing.T) {
	results, err := Query(`//div/@id | //div/@id`, `<main><div id="one"></div><div id="two"></div></main>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want one attribute per div: %#v", len(results), results)
	}
	wantValues := []string{"one", "two"}
	wantPaths := []string{"/#document/html/body/main/div[1]/id", "/#document/html/body/main/div[2]/id"}
	for i, result := range results {
		if result.NodeType != 2 || result.NodeName != "id" {
			t.Errorf("result %d node = %#v, want id attribute", i, result)
		}
		if result.Value != wantValues[i] {
			t.Errorf("result %d value = %q, want %q", i, result.Value, wantValues[i])
		}
		if result.Path != wantPaths[i] {
			t.Errorf("result %d path = %q, want %q", i, result.Path, wantPaths[i])
		}
	}
}
