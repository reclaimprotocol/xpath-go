package main

import (
	"encoding/json"
	"fmt"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test data with two divs
	html := `<html><body><div class="red">A</div><div class="blue">B</div><div id="test" class="active">C</div></body></html>`

	// Test cases to debug
	testCases := []string{
		"//div[@class='red']",                  // Should work
		"//div[@class='blue']",                 // Should work
		"//div[@class='red' or @class='blue']", // Should return both A and B
		"//div[@id and @class]",                // Should return C
	}

	for i, xpathExpr := range testCases {
		fmt.Printf("\n=== Test %d: %s ===\n", i+1, xpathExpr)

		results, err := xpath.Query(xpathExpr, html)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			continue
		}

		fmt.Printf("Count: %d\n", len(results))
		jsonOutput, _ := json.MarshalIndent(results, "", "  ")
		fmt.Printf("Results: %s\n", string(jsonOutput))
	}
}
