package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div></div><div> </div><div>Content</div></body></html>`
	
	fmt.Println("Testing Whitespace Preservation")
	fmt.Println("HTML:", html)
	fmt.Println()
	
	// Test all divs to see their text content
	results, err := xpath.Query("//div", html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("All divs (%d results):\n", len(results))
	for i, r := range results {
		fmt.Printf("%d. Text: '%s' (len=%d, bytes=%v)\n", i+1, r.TextContent, len(r.TextContent), []byte(r.TextContent))
	}
	
	fmt.Println("\n--- Debug normalize-space function ---")
	
	// Test normalize-space individually
	results2, _ := xpath.Query("//div", html)
	for i, r := range results2 {
		// Manually test normalize-space logic
		normalizedText := normalizeSpace(r.TextContent)
		fmt.Printf("Div %d: original='%s' -> normalized='%s' -> isEmpty=%v\n", 
			i+1, r.TextContent, normalizedText, normalizedText == "")
	}
}

func normalizeSpace(text string) string {
	// Simple normalize-space implementation to test
	fields := []string{}
	word := ""
	for _, char := range text {
		if char == ' ' || char == '\t' || char == '\n' || char == '\r' {
			if word != "" {
				fields = append(fields, word)
				word = ""
			}
		} else {
			word += string(char)
		}
	}
	if word != "" {
		fields = append(fields, word)
	}
	
	result := ""
	for i, field := range fields {
		if i > 0 {
			result += " "
		}
		result += field
	}
	return result
}