package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div></div><div>Content</div><div><span></span></div></body></html>`
	xpath1 := `//div[not(node())]`
	
	fmt.Println("Testing Empty Elements")
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
		fmt.Printf("   Value: '%s'\n", r.Value)
		fmt.Printf("   Path: %s\n", r.Path)
		if len(r.Attributes) > 0 {
			fmt.Printf("   Attributes: %v\n", r.Attributes)
		}
	}
	
	fmt.Println("\n--- Expected Results ---")
	fmt.Println("JavaScript finds 1 node: the first empty <div></div>")
	fmt.Println("The XPath //div[not(node())] should match div elements with NO child nodes")
	fmt.Println("The third div has a <span> child, so it should NOT match")
}