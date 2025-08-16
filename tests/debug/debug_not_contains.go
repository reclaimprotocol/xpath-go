package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Focus on the not(contains()) issue
	html := `<html><body><div class='primary active large'>Item 1</div><div class='secondary active'>Item 2</div><div class='primary inactive'>Item 3</div></body></html>`

	fmt.Println("=== Testing not(contains()) Issue ===")
	fmt.Println("HTML:")
	fmt.Println("- Item 1: class='primary active large' (no 'inactive')")
	fmt.Println("- Item 2: class='secondary active' (no 'inactive')")
	fmt.Println("- Item 3: class='primary inactive' (HAS 'inactive')")
	fmt.Println()

	tests := []struct {
		query    string
		expected []string
		name     string
	}{
		{
			"//div[contains(@class, 'inactive')]",
			[]string{"Item 3"},
			"Has 'inactive'",
		},
		{
			"//div[not(contains(@class, 'inactive'))]",
			[]string{"Item 1", "Item 2"},
			"Does NOT have 'inactive'",
		},
	}

	for _, test := range tests {
		results, err := xpath.Query(test.query, html)
		if err != nil {
			fmt.Printf("❌ %s: ERROR - %v\n", test.name, err)
			continue
		}

		status := "✅"
		if len(results) != len(test.expected) {
			status = "❌"
		}

		fmt.Printf("%s %s: %d results (expected %d)\n", status, test.name, len(results), len(test.expected))

		fmt.Println("  Found:")
		for i, result := range results {
			fmt.Printf("    %d. %s\n", i+1, result.TextContent)
		}

		if len(results) != len(test.expected) {
			fmt.Printf("  Expected: %v\n", test.expected)
		}
		fmt.Println()
	}

	fmt.Println("🔍 The issue: not(contains(@class, 'inactive')) is behaving incorrectly!")
	fmt.Println("It should return elements that DON'T contain 'inactive', but it's doing the opposite.")
}
