package xpath

import "testing"

func TestIncludeLocationFalsePreservesValuesAcrossOutputModes(t *testing.T) {
	for _, output := range []string{"nodes", "values", "paths"} {
		t.Run(output, func(t *testing.T) {
			results, err := QueryWithOptions(`//p`, `<p>text</p>`, Options{IncludeLocation: false, OutputFormat: output})
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != 1 {
				t.Fatalf("got %#v, want one result", results)
			}
			result := results[0]
			if result.StartLocation != 0 || result.EndLocation != 0 || result.ContentStart != 0 || result.ContentEnd != 0 {
				t.Fatalf("IncludeLocation=false leaked offsets: %#v", result)
			}
			switch output {
			case "nodes":
				if result.Value != `<p>text</p>` {
					t.Fatalf("nodes value = %q, want full source", result.Value)
				}
			case "values":
				if result.Value != "text" {
					t.Fatalf("values value = %q, want text", result.Value)
				}
			case "paths":
				if result.Value == "" || result.Value != result.Path {
					t.Fatalf("paths value = %q path = %q, want matching non-empty path", result.Value, result.Path)
				}
			}
		})
	}
}

func TestContentsOnlyWithoutLocationsKeepsContentValue(t *testing.T) {
	results, err := QueryWithOptions(`//p`, `<p>text</p>`, Options{IncludeLocation: false, ContentsOnly: true, OutputFormat: "nodes"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Value != "text" {
		t.Fatalf("contents-only result = %#v, want text value", results)
	}
	result := results[0]
	if result.StartLocation != 0 || result.EndLocation != 0 || result.ContentStart != 0 || result.ContentEnd != 0 {
		t.Fatalf("IncludeLocation=false leaked offsets: %#v", result)
	}
}
