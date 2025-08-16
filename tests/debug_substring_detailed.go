package main

import (
	"fmt"
	"log"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func main() {
	
	fmt.Println("=== DETAILED SUBSTRING DEBUGGING ===")
	fmt.Println()
	
	// Test each paragraph individually to see what substring returns
	texts := []string{"ShortText", "VeryLongTextHere", "Mid"}
	
	for i, text := range texts {
		fmt.Printf("=== Paragraph %d: '%s' ===\n", i+1, text)
		
		// Test just the substring function (without comparison)
		html := fmt.Sprintf(`<html><body><p>%s</p></body></html>`, text)
		
		// Test individual components
		result1, err := xpath.Query("//p[string-length(text())]", html)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("string-length() works: %d results\n", len(result1))
		
		// Test arithmetic
		result2, err := xpath.Query("//p[string-length(text()) - 3]", html)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("string-length() - 3 works: %d results\n", len(result2))
		
		// Test just substring (without comparison)
		result3, err := xpath.Query("//p[substring(text(), string-length(text()) - 3)]", html)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("substring() alone: %d results\n", len(result3))
		
		// Test the full comparison
		result4, err := xpath.Query("//p[substring(text(), string-length(text()) - 3) = 'Text']", html)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Full comparison: %d results\n", len(result4))
		
		fmt.Println()
	}
	
	fmt.Println("If all return 1 result, the issue is that our substring is always matching 'Text'")
	fmt.Println("If some return 0, the issue is in specific cases")
}