package main

import (
	"fmt"
	"log"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func main() {
	fmt.Println("=== FINAL SUBSTRING DEBUG ===")
	fmt.Println()
	
	// Test the exact failing case
	html := `<html><body><p>ShortText</p><p>VeryLongTextHere</p><p>Mid</p></body></html>`
	xpathExpr := `//p[substring(text(), string-length(text()) - 3) = 'Text']`
	
	fmt.Printf("XPath: %s\n", xpathExpr)
	fmt.Printf("HTML: %s\n", html)
	fmt.Println()
	
	result, err := xpath.Query(xpathExpr, html)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Results count: %d\n", len(result))
	for i, node := range result {
		fmt.Printf("Result %d: <%s>%s</%s>\n", i+1, node.NodeName, node.TextContent, node.NodeName)
	}
	
	fmt.Println()
	fmt.Println("Expected: Only 'ShortText' should match")
	fmt.Println("Analysis:")
	
	testCases := []string{"ShortText", "VeryLongTextHere", "Mid"}
	for _, text := range testCases {
		length := len(text)
		startPos := length - 3
		fmt.Printf("'%s' (len=%d): startPos=%d", text, length, startPos)
		
		if startPos >= 1 && startPos <= length {
			goStart := startPos - 1
			if goStart >= 0 && goStart < len(text) {
				substring := text[goStart:]
				fmt.Printf(" -> substring='%s' -> matches 'Text'? %t", substring, substring == "Text")
			}
		} else {
			fmt.Printf(" -> invalid position")
		}
		fmt.Println()
	}
}