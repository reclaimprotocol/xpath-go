package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><article><section><p id='target'>Content</p></section></article></body></html>`
	xpath1 := `//p[@id='target']/ancestor-or-self::*`
	
	fmt.Println("Testing Ancestor-or-self Axis")
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
		fmt.Printf("   Path: %s\n", r.Path)
		if len(r.Attributes) > 0 {
			fmt.Printf("   Attributes: %v\n", r.Attributes)
		}
	}
	
	fmt.Println("\n--- Expected Results ---")
	fmt.Println("JavaScript finds 5 nodes (from target up to html):")
	fmt.Println("1. html")
	fmt.Println("2. body")
	fmt.Println("3. article")
	fmt.Println("4. section")
	fmt.Println("5. p[@id='target']")
	fmt.Println("\nThe ancestor-or-self axis should return the node itself and all its ancestors")
}