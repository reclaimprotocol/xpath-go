package main

import (
	"fmt"
	"os"

	"github.com/reclaimprotocol/xpath-go/internal/evaluator"
)

func main() {
	html := `<html><body><article><header><h1>Title</h1></header><section><p>Content</p><aside><p>Sidebar</p></aside></section></article></body></html>`
	
	eval := evaluator.NewEvaluator()
	
	// First, let's see all p elements in section
	results1, err := eval.Evaluate(`//article//section//p`, html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("All p elements in section: %d\n", len(results1))
	for i, result := range results1 {
		fmt.Printf("  Result %d: '%s'\n", i+1, result.TextContent)
	}
	fmt.Println()

	// Now let's test the not(ancestor::aside) predicate
	results2, err := eval.Evaluate(`//article//section//p[not(ancestor::aside)]`, html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("P elements NOT in aside: %d\n", len(results2))
	for i, result := range results2 {
		fmt.Printf("  Result %d: '%s'\n", i+1, result.TextContent)
	}
}