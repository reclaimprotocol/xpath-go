package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html1 := `<html><body><div class='primary active large'>Item 1</div><div class='secondary active'>Item 2</div><div class='primary inactive'>Item 3</div></body></html>`
	xpath1 := `//div[contains(@class, 'primary') and contains(@class, 'active') and not(contains(@class, 'inactive'))]`
	
	fmt.Println("🔍 DEBUGGING COMPLEX AND NOT EXPRESSION")
	fmt.Println("==========================================")
	fmt.Printf("HTML: %s\n", html1)
	fmt.Printf("XPath: %s\n", xpath1)
	fmt.Println()
	
	xpath.EnableTrace()
	defer xpath.DisableTrace()
	
	results, err := xpath.Query(xpath1, html1)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("Found %d matches:\n", len(results))
		for i, result := range results {
			fmt.Printf("  %d. %s: '%s'\n", i+1, result.NodeName, result.TextContent)
		}
	}
	
	fmt.Println("\n🔍 DEBUGGING DOCUMENT STRUCTURE VALIDATION")
	fmt.Println("===========================================")
	
	html2 := `<html><head><title>Test</title><meta charset='utf-8'/></head><body><main><h1>Title</h1><p>Content</p></main></body></html>`
	xpath2 := `/html[head/title and head/meta[@charset] and body/main]`
	
	fmt.Printf("HTML: %s\n", html2)
	fmt.Printf("XPath: %s\n", xpath2)
	fmt.Println()
	
	results2, err2 := xpath.Query(xpath2, html2)
	if err2 != nil {
		fmt.Printf("ERROR: %v\n", err2)
	} else {
		fmt.Printf("Found %d matches:\n", len(results2))
		for i, result := range results2 {
			fmt.Printf("  %d. %s: '%s'\n", i+1, result.NodeName, result.TextContent)
		}
	}
}