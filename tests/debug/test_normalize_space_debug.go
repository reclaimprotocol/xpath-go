package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`
	
	fmt.Println("Testing normalize-space edge case")
	fmt.Println("HTML:", html)
	fmt.Println()
	
	// Test normalize-space with space-only div specifically
	fmt.Println("Testing if normalize-space(' ') equals ''")
	
	// Test 1: Just the second div with normalize-space
	fmt.Println("1. //div[position()=2][normalize-space(text())='']")
	results1, _ := xpath.Query("//div[position()=2][normalize-space(text())='']", html)
	fmt.Printf("Found %d results:\n", len(results1))
	for i, r := range results1 {
		fmt.Printf("  %d. Text: '%s' (len=%d)\n", i+1, r.TextContent, len(r.TextContent))
	}
	
	// Test 2: Just the second div with not(*)
	fmt.Println("\n2. //div[position()=2][not(*)]")
	results2, _ := xpath.Query("//div[position()=2][not(*)]", html)
	fmt.Printf("Found %d results:\n", len(results2))
	for i, r := range results2 {
		fmt.Printf("  %d. Text: '%s' (len=%d)\n", i+1, r.TextContent, len(r.TextContent))
	}
	
	// Test 3: Second div with both conditions
	fmt.Println("\n3. //div[position()=2][normalize-space(text())='' and not(*)]")
	results3, _ := xpath.Query("//div[position()=2][normalize-space(text())='' and not(*)]", html)
	fmt.Printf("Found %d results:\n", len(results3))
	for i, r := range results3 {
		fmt.Printf("  %d. Text: '%s' (len=%d)\n", i+1, r.TextContent, len(r.TextContent))
	}
	
	// Test 4: Direct test of normalize-space function
	fmt.Println("\n4. Testing normalize-space directly")
	results4, _ := xpath.Query("//div[position()=2]/text()", html)
	if len(results4) > 0 {
		fmt.Printf("Second div text node: '%s' (len=%d)\n", results4[0].TextContent, len(results4[0].TextContent))
	}
}