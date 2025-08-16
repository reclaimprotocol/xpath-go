package main

import (
	"fmt"
	"log"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func main() {
	fmt.Println("=== ROOT CAUSE ANALYSIS: SUBSTRING FUNCTION ===")
	fmt.Println()
	
	// Test cases from the failing test
	testCases := []struct {
		text     string
		expected bool
	}{
		{"ShortText", true},        // substring('ShortText', 6) = 'Text' ✓
		{"VeryLongTextHere", false}, // substring('VeryLongTextHere', 13) = 'Here' ≠ 'Text'
		{"Mid", false},             // substring('Mid', 0) = invalid or 'd' ≠ 'Text'
	}
	
	for _, tc := range testCases {
		fmt.Printf("=== Testing: '%s' ===\n", tc.text)
		
		html := fmt.Sprintf(`<html><body><p>%s</p></body></html>`, tc.text)
		
		// Test what our implementation returns
		result, err := xpath.Query("//p[substring(text(), string-length(text()) - 3) = 'Text']", html)
		if err != nil {
			log.Fatal(err)
		}
		
		actual := len(result) > 0
		fmt.Printf("Expected: %t, Got: %t", tc.expected, actual)
		
		if actual != tc.expected {
			fmt.Printf(" ❌ WRONG")
		} else {
			fmt.Printf(" ✅ CORRECT")
		}
		fmt.Println()
		
		// Calculate what the substring should actually be
		textLen := len(tc.text)
		startPos := textLen - 3  // XPath 1-based position
		
		fmt.Printf("Text: '%s' (length: %d)\n", tc.text, textLen)
		fmt.Printf("Start position: %d - 3 = %d\n", textLen, startPos)
		
		if startPos >= 1 {
			// Convert to 0-based for Go
			goStart := startPos - 1
			if goStart >= 0 && goStart < len(tc.text) {
				substring := tc.text[goStart:]
				fmt.Printf("Substring from pos %d: '%s'\n", startPos, substring)
				fmt.Printf("'%s' == 'Text'? %t\n", substring, substring == "Text")
			} else {
				fmt.Printf("Invalid Go start position: %d\n", goStart)
			}
		} else {
			fmt.Printf("Invalid XPath start position: %d\n", startPos)
		}
		fmt.Println()
	}
	
	fmt.Println("Root cause analysis:")
	fmt.Println("- If all show 'Got: true', the substring function always returns 'Text'")
	fmt.Println("- If results are wrong but calculations are right, the issue is in argument parsing")
	fmt.Println("- If calculations are wrong, the issue is in the XPath substring specification")
}