package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test the exact failing count case
	html := `<html><body><ul><li>A</li><li>B</li><li>C</li></ul></body></html>`
	
	query := `//ul[count(li)=3]`
	
	fmt.Println("=== Testing count() function ===")
	fmt.Printf("HTML: %s\n", html)
	fmt.Printf("Query: %s\n", query)
	fmt.Println()
	
	fmt.Println("Expected: Should return 1 result (the ul element)")
	fmt.Println("Because the ul contains exactly 3 li elements")
	fmt.Println()
	
	// Test the query
	results, err := xpath.Query(query, html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	
	fmt.Printf("Go results: %d\n", len(results))
	for i, result := range results {
		fmt.Printf("  %d. %s (tag: %s)\n", i+1, result.TextContent, result.NodeName)
	}
	
	if len(results) == 1 {
		fmt.Println("✅ SUCCESS! count() function works correctly")
	} else {
		fmt.Println("❌ FAILED - count() function not working correctly")
		
		// Debug: Check if we can find li elements
		fmt.Println("\n=== Debug: Check li elements ===")
		liResults, err := xpath.Query("//li", html)
		if err != nil {
			fmt.Printf("ERROR finding li elements: %v\n", err)
		} else {
			fmt.Printf("Found %d li elements:\n", len(liResults))
			for i, li := range liResults {
				fmt.Printf("  %d. %s\n", i+1, li.TextContent)
			}
		}
		
		// Debug: Check ul element
		fmt.Println("\n=== Debug: Check ul elements ===")
		ulResults, err := xpath.Query("//ul", html)
		if err != nil {
			fmt.Printf("ERROR finding ul elements: %v\n", err)
		} else {
			fmt.Printf("Found %d ul elements:\n", len(ulResults))
			for i, ul := range ulResults {
				fmt.Printf("  %d. %s\n", i+1, ul.TextContent)
			}
		}
	}
}