package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><p>Short text here</p><p>This is a much longer paragraph with more than ten characters after normalization</p><p>   Whitespace   </p></body></html>`
	xpath1 := `//p[string-length(normalize-space(text())) > 10]`
	
	fmt.Println("Testing String-length with normalize-space combination")
	fmt.Println("HTML:", html)
	fmt.Println("XPath:", xpath1)
	fmt.Println()
	
	results, err := xpath.Query(xpath1, html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Found %d results:\n", len(results))
	for i, r := range results {
		fmt.Printf("%d. NodeName: %s, Text: '%s'\n", i+1, r.NodeName, r.TextContent)
		fmt.Printf("   Text Length: %d\n", len(r.TextContent))
		fmt.Printf("   Path: %s\n", r.Path)
	}
	
	fmt.Println("\n--- Expected Results ---")
	fmt.Println("JavaScript finds 2 nodes:")
	fmt.Println("1. The first p with 'Short text here' (15 chars)")
	fmt.Println("2. The second p with long text (>10 chars after normalize-space)")
	fmt.Println("The third p should be excluded because normalize-space('   Whitespace   ') = 'Whitespace' (10 chars, not > 10)")
}