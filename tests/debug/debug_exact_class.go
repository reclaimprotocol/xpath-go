package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Use the EXACT test data from the comprehensive test
	html := `<html><body><div class='primary active large'>Item 1</div><div class='secondary active'>Item 2</div><div class='primary inactive'>Item 3</div></body></html>`
	
	query := `//div[contains(@class, 'primary') and contains(@class, 'active') and not(contains(@class, 'inactive'))]`
	
	fmt.Println("=== Exact Class List Test (From Comprehensive Suite) ===")
	fmt.Printf("HTML: %s\n", html)
	fmt.Printf("Query: %s\n", query)
	fmt.Println()
	
	fmt.Println("Element Analysis:")
	fmt.Println("1. Item 1: class='primary active large'")
	fmt.Println("   - contains(class, 'primary'): true ✅")
	fmt.Println("   - contains(class, 'active'): true ✅") 
	fmt.Println("   - not(contains(class, 'inactive')): true ✅")
	fmt.Println("   → Should MATCH ✅")
	fmt.Println()
	
	fmt.Println("2. Item 2: class='secondary active'")
	fmt.Println("   - contains(class, 'primary'): false ❌")
	fmt.Println("   - contains(class, 'active'): true ✅")
	fmt.Println("   - not(contains(class, 'inactive')): true ✅") 
	fmt.Println("   → Should NOT match (fails first condition)")
	fmt.Println()
	
	fmt.Println("3. Item 3: class='primary inactive'")
	fmt.Println("   - contains(class, 'primary'): true ✅")
	fmt.Println("   - contains(class, 'active'): false ❌")
	fmt.Println("   - not(contains(class, 'inactive')): false ❌")
	fmt.Println("   → Should NOT match (fails second and third conditions)")
	fmt.Println()
	
	fmt.Println("Expected: 1 result (Item 1 only)")
	fmt.Println()
	
	// Test the query
	results, err := xpath.Query(query, html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	
	fmt.Printf("Go results: %d\n", len(results))
	for i, result := range results {
		fmt.Printf("  %d. %s\n", i+1, result.TextContent)
	}
	
	if len(results) == 1 && results[0].TextContent == "Item 1" {
		fmt.Println("✅ SUCCESS!")
	} else {
		fmt.Println("❌ FAILED - Go should return exactly 1 result (Item 1)")
		
		// Debug individual components
		fmt.Println("\n=== Debugging Components ===")
		
		components := []string{
			"//div[contains(@class, 'primary')]",
			"//div[contains(@class, 'active')]",
			"//div[contains(@class, 'inactive')]",
			"//div[not(contains(@class, 'inactive'))]",
			"//div[contains(@class, 'primary') and contains(@class, 'active')]",
		}
		
		for _, comp := range components {
			compResults, err := xpath.Query(comp, html)
			if err != nil {
				fmt.Printf("  %s: ERROR - %v\n", comp, err)
			} else {
				fmt.Printf("  %s: %d results\n", comp, len(compResults))
				for j, res := range compResults {
					fmt.Printf("    %d. %s\n", j+1, res.TextContent)
				}
			}
		}
	}
}