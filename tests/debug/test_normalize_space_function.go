package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`
	
	fmt.Println("Testing normalize-space function behavior")
	fmt.Println("HTML:", html)
	fmt.Println()
	
	// Test normalize-space on each div individually
	for i := 1; i <= 4; i++ {
		fmt.Printf("Div %d:\n", i)
		
		// Get the div
		divQuery := fmt.Sprintf("//div[position()=%d]", i)
		divResults, _ := xpath.Query(divQuery, html)
		if len(divResults) > 0 {
			fmt.Printf("  Raw text: '%s' (len=%d)\n", divResults[0].TextContent, len(divResults[0].TextContent))
		}
		
		// Test normalize-space separately
		normalizeQuery := fmt.Sprintf("//div[position()=%d][normalize-space(text())='']", i)
		normalizeResults, _ := xpath.Query(normalizeQuery, html)
		fmt.Printf("  normalize-space(text())='': %s\n", boolToString(len(normalizeResults) > 0))
		
		// Test not(*) separately  
		notQuery := fmt.Sprintf("//div[position()=%d][not(*)]", i)
		notResults, _ := xpath.Query(notQuery, html)
		fmt.Printf("  not(*): %s\n", boolToString(len(notResults) > 0))
		
		// Test combined
		combinedQuery := fmt.Sprintf("//div[position()=%d][normalize-space(text())='' and not(*)]", i)
		combinedResults, _ := xpath.Query(combinedQuery, html)
		fmt.Printf("  combined: %s\n", boolToString(len(combinedResults) > 0))
		
		fmt.Println()
	}
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}