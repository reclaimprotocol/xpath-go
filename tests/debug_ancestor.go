package main

import (
	"fmt"
	"os"

	"github.com/reclaimprotocol/xpath-go/internal/evaluator"
)

func main() {
	html := `<html><body><article><header><h1>Title</h1></header><section><p>Content</p><aside><p>Sidebar</p></aside></section></article></body></html>`
	xpath := `//article//section//p[not(ancestor::aside)]`

	eval := evaluator.NewEvaluator()
	results, err := eval.Evaluate(xpath, html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Results count: %d\n", len(results))
	for i, result := range results {
		fmt.Printf("Result %d: %s\n", i+1, result.TextContent)
	}
}
