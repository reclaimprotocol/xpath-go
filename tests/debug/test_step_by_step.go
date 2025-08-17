package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`
	
	fmt.Println("Step-by-step debugging")
	fmt.Println("HTML:", html)
	fmt.Println()
	
	// Step 1: Test base selector
	fmt.Println("Step 1: Base selector //div")
	results1, _ := xpath.Query("//div", html)
	fmt.Printf("Found %d divs:\n", len(results1))
	for i, r := range results1 {
		fmt.Printf("  %d. Text: '%s' (len=%d)\n", i+1, r.TextContent, len(r.TextContent))
	}
	
	// Step 2: Test normalize-space condition only
	fmt.Println("\nStep 2: //div[normalize-space(text())='']")
	results2, _ := xpath.Query("//div[normalize-space(text())='']", html)
	fmt.Printf("Found %d divs:\n", len(results2))
	for i, r := range results2 {
		fmt.Printf("  %d. Text: '%s' (len=%d)\n", i+1, r.TextContent, len(r.TextContent))
	}
	
	// Step 3: Test not(*) condition only
	fmt.Println("\nStep 3: //div[not(*)]")
	results3, _ := xpath.Query("//div[not(*)]", html)
	fmt.Printf("Found %d divs:\n", len(results3))
	for i, r := range results3 {
		fmt.Printf("  %d. Text: '%s' (len=%d)\n", i+1, r.TextContent, len(r.TextContent))
	}
	
	// Step 4: Test the full combined expression
	fmt.Println("\nStep 4: //div[normalize-space(text())='' and not(*)]")
	results4, _ := xpath.Query("//div[normalize-space(text())='' and not(*)]", html)
	fmt.Printf("Found %d divs:\n", len(results4))
	for i, r := range results4 {
		fmt.Printf("  %d. Text: '%s' (len=%d)\n", i+1, r.TextContent, len(r.TextContent))
	}
}