package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div></div><div> </div><div>Content</div><div><span></span></div></body></html>`
	xpath1 := `//div[normalize-space(text())='' and not(*)]`
	
	fmt.Println("Testing Empty vs Non-empty Elements")
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
		fmt.Printf("   Text Bytes: %v\n", []byte(r.TextContent))
		fmt.Printf("   Path: %s\n", r.Path)
	}
	
	fmt.Println("\n--- Expected Results ---")
	fmt.Println("JavaScript finds 2 nodes:")
	fmt.Println("1. The first empty div: <div></div>")
	fmt.Println("2. The second div with only space: <div> </div>")
	fmt.Println("The XPath should match divs where:")
	fmt.Println("- normalize-space(text()) = '' (empty after normalization)")
	fmt.Println("- not(*) (no child elements)")
	
	// Test components separately
	fmt.Println("\n--- Debug Components ---")
	
	// Test normalize-space condition only
	results1, _ := xpath.Query("//div[normalize-space(text())='']", html)
	fmt.Printf("normalize-space(text())='' finds %d results:\n", len(results1))
	for i, r := range results1 {
		fmt.Printf("  %d. '%s' (len=%d)\n", i+1, r.TextContent, len(r.TextContent))
	}
	
	// Test not(*) condition only  
	results2, _ := xpath.Query("//div[not(*)]", html)
	fmt.Printf("not(*) finds %d results:\n", len(results2))
	for i, r := range results2 {
		fmt.Printf("  %d. '%s'\n", i+1, r.TextContent)
	}
}