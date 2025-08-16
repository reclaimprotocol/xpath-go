package main

import (
	"encoding/json"
	"fmt"
	"log"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func main() {
	fmt.Println("=== ACTUAL EVALUATION DEBUG ===")
	fmt.Println()
	
	// Test each paragraph individually
	testCases := []struct {
		name string
		html string
	}{
		{"ShortText", `<html><body><p>ShortText</p></body></html>`},
		{"VeryLongTextHere", `<html><body><p>VeryLongTextHere</p></body></html>`},
		{"Mid", `<html><body><p>Mid</p></body></html>`},
	}
	
	xpathExpr := `//p[substring(text(), string-length(text()) - 3) = 'Text']`
	
	for _, tc := range testCases {
		fmt.Printf("=== Testing: %s ===\n", tc.name)
		
		result, err := xpath.Query(xpathExpr, tc.html)
		if err != nil {
			log.Printf("Error for %s: %v", tc.name, err)
			continue
		}
		
		fmt.Printf("Results count: %d\n", len(result))
		
		if len(result) > 0 {
			resultJson, _ := json.MarshalIndent(result[0], "", "  ")
			fmt.Printf("Result: %s\n", resultJson)
			fmt.Printf("✅ MATCHED (but should it?)\n")
		} else {
			fmt.Printf("❌ NO MATCH\n")
		}
		
		// Expected results
		switch tc.name {
		case "ShortText":
			fmt.Printf("Expected: SHOULD match (substring('ShortText', 6) = 'Text')\n")
		case "VeryLongTextHere":
			fmt.Printf("Expected: should NOT match (substring('VeryLongTextHere', 13) = 'Here' ≠ 'Text')\n")
		case "Mid":
			fmt.Printf("Expected: should NOT match (substring('Mid', 0) = 'd' ≠ 'Text')\n")
		}
		fmt.Println()
	}
}