package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div class='primary active'>Item 1</div><div class='secondary active'>Item 2</div><div class='primary inactive'>Item 3</div></body></html>`
	xpath1 := `//div[contains(@class, 'primary') and contains(@class, 'active') and not(contains(@class, 'inactive'))]`
	
	fmt.Println("Testing Class List Manipulation")
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
		fmt.Printf("   Class: %s\n", r.Attributes["class"])
		fmt.Printf("   Path: %s\n", r.Path)
	}
	
	fmt.Println("\n--- Expected Results ---")
	fmt.Println("JavaScript finds 1 node:")
	fmt.Println("1. Item 1 (has 'primary' AND 'active' AND NOT 'inactive')")
	fmt.Println("Item 2: has 'active' but not 'primary'")
	fmt.Println("Item 3: has 'primary' but also has 'inactive'")
	
	// Test components separately
	fmt.Println("\n--- Debug Components ---")
	
	// Test individual conditions
	results1, _ := xpath.Query("//div[contains(@class, 'primary')]", html)
	fmt.Printf("contains(@class, 'primary') finds %d results:\n", len(results1))
	for i, r := range results1 {
		fmt.Printf("  %d. '%s' (class='%s')\n", i+1, r.TextContent, r.Attributes["class"])
	}
	
	results2, _ := xpath.Query("//div[contains(@class, 'active')]", html)
	fmt.Printf("contains(@class, 'active') finds %d results:\n", len(results2))
	for i, r := range results2 {
		fmt.Printf("  %d. '%s' (class='%s')\n", i+1, r.TextContent, r.Attributes["class"])
	}
	
	results3, _ := xpath.Query("//div[not(contains(@class, 'inactive'))]", html)
	fmt.Printf("not(contains(@class, 'inactive')) finds %d results:\n", len(results3))
	for i, r := range results3 {
		fmt.Printf("  %d. '%s' (class='%s')\n", i+1, r.TextContent, r.Attributes["class"])
	}
}