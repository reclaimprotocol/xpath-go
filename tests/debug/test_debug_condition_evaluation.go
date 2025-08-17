package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`
	
	fmt.Println("Debug condition evaluation for second div")
	fmt.Println("HTML:", html)
	fmt.Println()
	
	// Test parts of the failing expression separately
	tests := []string{
		"//div[position()=2]",
		"//div[normalize-space(text())='']",
		"//div[not(*)]", 
		"//div[normalize-space(text())='' and not(*)]",
		"//div[position()=2 and normalize-space(text())='']",
		"//div[position()=2 and not(*)]",
		"//div[position()=2 and normalize-space(text())='' and not(*)]",
	}
	
	for i, test := range tests {
		fmt.Printf("%d. %s\n", i+1, test)
		results, err := xpath.Query(test, html)
		if err != nil {
			fmt.Printf("   ERROR: %v\n", err)
		} else {
			fmt.Printf("   Found %d results\n", len(results))
			for j, r := range results {
				fmt.Printf("     %d. Text: '%s' (len=%d)\n", j+1, r.TextContent, len(r.TextContent))
			}
		}
		fmt.Println()
	}
}