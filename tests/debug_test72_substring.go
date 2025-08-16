package main

import (
	"fmt"
	"log"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><body><p>ShortText</p><p>VeryLongTextHere</p><p>Mid</p></body></html>`

	fmt.Println("=== DEBUGGING TEST 72 - SUBSTRING WITH STRING-LENGTH ===")
	fmt.Println()

	// Test the failing expression
	result, err := xpath.Query("//p[substring(text(), string-length(text()) - 3) = 'Text']", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Go results: %d\n", len(result))
	for i, r := range result {
		fmt.Printf("  Result %d: '%s'\n", i+1, r.TextContent)
	}

	// Test each paragraph individually to see what the substring returns
	fmt.Println("\nTesting each paragraph individually:")

	texts := []string{"ShortText", "VeryLongTextHere", "Mid"}
	for i, text := range texts {
		fmt.Printf("\nParagraph %d: '%s' (length: %d)\n", i+1, text, len(text))

		// Calculate what string-length(text()) - 3 should be
		startPos := len(text) - 3
		fmt.Printf("  string-length(text()) - 3 = %d - 3 = %d\n", len(text), startPos)

		// What should substring return?
		if startPos >= 1 && startPos <= len(text) {
			// XPath uses 1-based indexing
			goStartPos := startPos - 1 // Convert to 0-based for Go
			if goStartPos >= 0 && goStartPos < len(text) {
				substring := text[goStartPos:]
				fmt.Printf("  substring(text(), %d) should return: '%s'\n", startPos, substring)
				fmt.Printf("  Does '%s' = 'Text'? %t\n", substring, substring == "Text")
			}
		} else {
			fmt.Printf("  substring start position %d is out of bounds\n", startPos)
		}
	}

	fmt.Println("\nExpected behavior:")
	fmt.Println("- 'ShortText': substring from pos 6 = 'ext' ≠ 'Text'")
	fmt.Println("- 'VeryLongTextHere': substring from pos 14 = 'ere' ≠ 'Text'")
	fmt.Println("- 'Mid': substring from pos 0 (invalid) = ???")
	fmt.Println("\nNone should match 'Text', so result should be 0, not 3!")
}
