package main

import (
	"fmt"
	"strings"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`
	
	// Get all divs and examine their properties
	results, err := xpath.Query("//div", html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	
	fmt.Printf("Found %d divs:\n", len(results))
	for i, result := range results {
		textContent := result.TextContent
		fmt.Printf("Div %d:\n", i+1)
		fmt.Printf("  Text: '%s'\n", textContent)
		fmt.Printf("  Text length: %d\n", len(textContent))
		fmt.Printf("  Text bytes: %v\n", []byte(textContent))
		fmt.Printf("  Trimmed: '%s'\n", strings.TrimSpace(textContent))
		fmt.Printf("  Normalized: '%s'\n", normalizeSpace(textContent))
		fmt.Printf("  Has child elements: %v\n", hasChildElements(result.Value))
		fmt.Println()
	}
}

func normalizeSpace(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

func hasChildElements(value string) bool {
	// Simple check for child elements
	return strings.Contains(value, "<") && strings.Contains(value, ">")
}