package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Use the exact HTML from the failing test case
	html := `<html><body><ul><li><span>Item 1</span><!-- comment --></li><li>Item 2</li><li><a href='#'>Item 3</a><span>Extra</span></li></ul></body></html>`
	query := `//li[span and not(a)]`
	
	fmt.Printf("Testing: %s\n", query)
	fmt.Printf("Expected: 1 result (first li)\n")
	fmt.Println()
	
	// Test the actual query
	results, err := xpath.Query(query, html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	
	fmt.Printf("Results: %d\n", len(results))
	
	if len(results) == 1 {
		fmt.Println("✅ SUCCESS! Fixed the list mixed content test!")
	} else {
		fmt.Println("❌ Still failing...")
		
		// Debug each li individually  
		for i := 1; i <= 3; i++ {
			fmt.Printf("\n--- Li %d Debug ---\n", i)
			
			// Check what children it has
			childResults, err := xpath.Query(fmt.Sprintf("(//li)[%d]/*", i), html)
			if err != nil {
				fmt.Printf("Error getting children: %v\n", err)
			} else {
				fmt.Printf("Child elements: %d\n", len(childResults))
				for j, child := range childResults {
					fmt.Printf("  Child %d: <%s>\n", j+1, child.NodeName)
				}
			}
		}
	}
}