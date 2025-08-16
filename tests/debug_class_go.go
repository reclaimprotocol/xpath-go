package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><body><div class='primary active large'>Item 1</div><div class='secondary active'>Item 2</div><div class='primary inactive'>Item 3</div></body></html>`
	
	fmt.Println("=== TESTING COMPLEX CLASS LOGIC WITH GO ===")
	fmt.Println()
	
	// Test each div individually first
	allDivs, err := xpath.Query("//div", htmlContent)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d divs total\n\n", len(allDivs))

	for i, div := range allDivs {
		fmt.Printf("Div %d: <%s class=\"%s\">%s</%s>\n", 
			i+1, div.NodeName, div.Attributes["class"], div.TextContent, div.NodeName)
		
		classAttr := div.Attributes["class"]
		fmt.Printf("  Class: '%s' (length: %d)\n", classAttr, len(classAttr))
		
		// Test individual conditions
		containsPrimary := strings.Contains(classAttr, "primary")
		containsActive := strings.Contains(classAttr, "active") 
		containsInactive := strings.Contains(classAttr, "inactive")
		
		fmt.Printf("  contains(class, 'primary'): %t\n", containsPrimary)
		fmt.Printf("  contains(class, 'active'): %t\n", containsActive)
		fmt.Printf("  contains(class, 'inactive'): %t\n", containsInactive)
		fmt.Printf("  not(contains(class, 'inactive')): %t\n", !containsInactive)
		
		shouldMatch := containsPrimary && containsActive && !containsInactive
		fmt.Printf("  Should match: %t\n", shouldMatch)
		fmt.Println()
	}

	// Now test the full XPath
	query := "//div[contains(@class, 'primary') and contains(@class, 'active') and not(contains(@class, 'inactive'))]"
	fmt.Printf("Full XPath: %s\n", query)
	
	results, err := xpath.Query(query, htmlContent)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Go found: %d results\n", len(results))
	for i, result := range results {
		fmt.Printf("  Result %d: <%s class=\"%s\">%s</%s>\n", 
			i+1, result.NodeName, result.Attributes["class"], result.TextContent, result.NodeName)
	}
}