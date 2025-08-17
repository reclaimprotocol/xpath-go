package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`
	
	fmt.Println("Testing simple and condition without position")
	fmt.Println("HTML:", html)
	fmt.Println()
	
	// Let's create a test case that should clearly show 2 results
	// First, test each condition separately
	fmt.Println("1. Individual conditions:")
	
	results1, _ := xpath.Query("//div[normalize-space(text())='']", html)
	fmt.Printf("   normalize-space(text())='': %d results\n", len(results1))
	for i, r := range results1 {
		fmt.Printf("     %d. Text: '%s' (len=%d)\n", i+1, r.TextContent, len(r.TextContent))
	}
	
	results2, _ := xpath.Query("//div[not(*)]", html)
	fmt.Printf("   not(*): %d results\n", len(results2))
	for i, r := range results2 {
		fmt.Printf("     %d. Text: '%s' (len=%d)\n", i+1, r.TextContent, len(r.TextContent))
	}
	
	fmt.Println("\n2. Combined condition:")
	results3, _ := xpath.Query("//div[normalize-space(text())='' and not(*)]", html)
	fmt.Printf("   Combined: %d results\n", len(results3))
	for i, r := range results3 {
		fmt.Printf("     %d. Text: '%s' (len=%d)\n", i+1, r.TextContent, len(r.TextContent))
	}
	
	fmt.Println("\n3. Expected result based on intersection:")
	fmt.Println("   The first condition matches divs 1, 2, 3")
	fmt.Println("   The second condition matches divs 1, 2, 4") 
	fmt.Println("   Intersection should be divs 1, 2 (empty and space-only)")
}