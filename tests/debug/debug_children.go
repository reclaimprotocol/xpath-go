package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`

	// Get all divs and examine their children
	results, err := xpath.Query("//div", html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	fmt.Printf("Found %d divs:\n", len(results))
	for i, result := range results {
		fmt.Printf("Div %d: text='%s'\n", i+1, result.TextContent)

		// Test our not(*) predicate directly
		notStarResults, err := xpath.Query(fmt.Sprintf("(//div)[%d][not(*)]", i+1), html)
		if err != nil {
			fmt.Printf("  not(*) test ERROR: %v\n", err)
		} else {
			fmt.Printf("  not(*) matches: %v\n", len(notStarResults) > 0)
		}

		normalizeResults, err := xpath.Query(fmt.Sprintf("(//div)[%d][normalize-space(text())='']", i+1), html)
		if err != nil {
			fmt.Printf("  normalize-space test ERROR: %v\n", err)
		} else {
			fmt.Printf("  normalize-space matches: %v\n", len(normalizeResults) > 0)
		}
		fmt.Println()
	}
}
