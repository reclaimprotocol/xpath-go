package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`
	
	fmt.Println("Testing not(*) predicate specifically")
	fmt.Println("HTML:", html)
	fmt.Println()
	
	// Test not(*) specifically
	results, err := xpath.Query("//div[not(*)]", html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Found %d results for //div[not(*)]:\n", len(results))
	for i, r := range results {
		fmt.Printf("%d. Text: '%s' (len=%d)\n", i+1, r.TextContent, len(r.TextContent))
	}
	
	fmt.Println("\nExpected:")
	fmt.Println("Should find 3 divs (all except the one with <span> child)")
	fmt.Println("1. <div></div> - no children")
	fmt.Println("2. <div> </div> - only text child (not element)")
	fmt.Println("3. <div>Content</div> - only text child (not element)")
	fmt.Println("Should NOT find: <div><span></span></div> - has element child")
	
	// Test just * to see what it matches
	fmt.Println("\n--- Testing * selector ---")
	results2, _ := xpath.Query("//div/*", html)
	fmt.Printf("//div/* finds %d results:\n", len(results2))
	for i, r := range results2 {
		fmt.Printf("%d. Element: %s\n", i+1, r.NodeName)
	}
}