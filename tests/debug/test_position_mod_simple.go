package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<div><span class="item">First</span><span>Middle</span><span class="item">Second</span><span class="item">Third</span></div>`
	
	fmt.Println("Testing position mod issue:")
	xpath.EnableTrace()
	
	// Test the failing case
	results, err := xpath.Query(`//span[@class='item'][position() mod 2 = 0]`, html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
	} else {
		fmt.Printf("Found %d matches:\n", len(results))
		for i, result := range results {
			fmt.Printf("  %d. %s: '%s'\n", i+1, result.NodeName, result.TextContent)
		}
	}
	
	xpath.DisableTrace()
}