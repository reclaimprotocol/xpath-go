package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go/internal/parser"
)

func main() {
	p := parser.NewParser()

	fmt.Println("Testing XPath parsing:")

	testCases := []string{
		"//div",
		"/html",
		"div",
		"//div[@id='test']",
	}

	for _, xpath := range testCases {
		fmt.Printf("\n--- Testing: %s ---\n", xpath)
		parsed, err := p.Parse(xpath)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
		} else {
			fmt.Printf("SUCCESS: IsAbsolute=%v, Steps=%d\n", parsed.IsAbsolute, len(parsed.Steps))
			for i, step := range parsed.Steps {
				fmt.Printf("  Step %d: Axis=%s, NodeTest=%s, Predicates=%d\n",
					i, step.Axis, step.NodeTest, len(step.Predicates))
			}
		}
	}
}
