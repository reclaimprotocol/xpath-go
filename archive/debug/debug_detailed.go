package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go/internal/parser"
)

func main() {
	p := parser.NewParser()
	
	// Let's manually trace through the tokenization and parsing
	expr := "//div"
	fmt.Printf("Testing detailed parsing of: %s\n", expr)
	
	// First, let's call Parse and catch the error to see what happens
	_, err := p.Parse(expr)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
	}
	
	// Let's also test a working case for comparison
	fmt.Printf("\nTesting working case: /html\n")
	parsed, err := p.Parse("/html")
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
	} else {
		fmt.Printf("SUCCESS: IsAbsolute=%v, Steps=%d\n", parsed.IsAbsolute, len(parsed.Steps))
	}
}