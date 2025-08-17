package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`
	
	fmt.Println("=== TRACING EVALUATION PATH ===")
	fmt.Println("HTML:", html)
	fmt.Println("XPath: //div[normalize-space(text())='' and not(*)]")
	fmt.Println()
	
	// This will trigger our debug logging throughout the evaluation pipeline
	results, err := xpath.Query("//div[normalize-space(text())='' and not(*)]", html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("\nFinal Results: %d matches\n", len(results))
	for i, result := range results {
		fmt.Printf("  %d. Text: '%s' (len=%d)\n", i+1, result.TextContent, len(result.TextContent))
	}
	
	fmt.Println("\nExpected: 2 matches (empty div and space-only div)")
}