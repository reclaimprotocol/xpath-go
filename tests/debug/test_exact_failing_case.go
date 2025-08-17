package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Exact test case from extended_testcases.json
	html := `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`
	xpath1 := `//div[normalize-space(text())='' and not(*)]`
	
	fmt.Println("Testing Exact Failing Case")
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
		fmt.Printf("%d. Text: '%s' (len=%d, bytes=%v)\n", i+1, r.TextContent, len(r.TextContent), []byte(r.TextContent))
		if len(r.Attributes) > 0 {
			fmt.Printf("   Attributes: %v\n", r.Attributes)
		}
	}
	
	fmt.Println("\n--- Expected by JavaScript ---")
	fmt.Println("Should find 2 results:")
	fmt.Println("1. First div (empty): <div></div>")
	fmt.Println("2. Second div (space only): <div> </div>")
	fmt.Println("Should NOT find:")
	fmt.Println("- Third div: <div><span></span></div> (has child element)")
	fmt.Println("- Fourth div: <div>Content</div> (has non-empty text after normalize-space)")
	
	// Test all divs raw
	fmt.Println("\n--- All Divs Raw ---")
	allDivs, _ := xpath.Query("//div", html)
	for i, div := range allDivs {
		fmt.Printf("Div %d: text='%s' (len=%d)\n", 
			i+1, div.TextContent, len(div.TextContent))
	}
}