package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<table><tbody><tr><td>Header 1</td><td>Header 2</td></tr><tr><td>Row 1 Col 1</td><td>Row 1 Col 2</td></tr><tr><td>Row 2 Col 1</td><td>Row 2 Col 2</td></tr></tbody></table>`
	xpath1 := `//tbody/tr[position()>1]/td[position()=1]`
	
	fmt.Println("Testing Complex Table Navigation with Position Predicates")
	fmt.Println("HTML:", html)
	fmt.Println("XPath:", xpath1)
	fmt.Println()
	
	results, err := xpath.Query(xpath1, html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Found %d results:\n", len(results))
	for i, r := range results {
		fmt.Printf("%d. NodeName: %s, Text: '%s'\n", i+1, r.NodeName, r.TextContent)
		fmt.Printf("   Path: %s\n", r.Path)
	}
	
	fmt.Println("\n--- Expected Results ---")
	fmt.Println("JavaScript finds 1 node:")
	fmt.Println("1. The first td in the second tr: 'Row 1 Col 1'")
	fmt.Println("")
	fmt.Println("Breakdown:")
	fmt.Println("- //tbody/tr[position()>1] should select tr elements after the first one")
	fmt.Println("- /td[position()=1] should select the first td in each selected tr")
	fmt.Println("- This should give us the first column of rows 2 and 3")
	
	// Test individual parts
	fmt.Println("\n--- Debug Individual Parts ---")
	
	// Test just the tr selection
	results1, _ := xpath.Query("//tbody/tr[position()>1]", html)
	fmt.Printf("//tbody/tr[position()>1] finds %d results:\n", len(results1))
	for i, r := range results1 {
		fmt.Printf("  %d. %s: '%s'\n", i+1, r.NodeName, r.TextContent)
	}
	
	// Test just position()=1 on all td
	results2, _ := xpath.Query("//td[position()=1]", html)
	fmt.Printf("//td[position()=1] finds %d results:\n", len(results2))
	for i, r := range results2 {
		fmt.Printf("  %d. %s: '%s'\n", i+1, r.NodeName, r.TextContent)
	}
}