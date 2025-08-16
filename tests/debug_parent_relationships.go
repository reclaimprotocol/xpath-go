package main

import (
	"fmt"
	"os"

	"github.com/reclaimprotocol/xpath-go/internal/evaluator"
	"github.com/reclaimprotocol/xpath-go/pkg/types"
	"github.com/reclaimprotocol/xpath-go/pkg/utils"
)

func main() {
	html := `<html><body><article><header><h1>Title</h1></header><section><p>Content</p><aside><p>Sidebar</p></aside></section></article></body></html>`
	
	// Parse the HTML and check node structure
	parser := utils.NewHTMLParser()
	document, err := parser.Parse(html)
	if err != nil {
		fmt.Printf("Error parsing: %v\n", err)
		os.Exit(1)
	}

	// Find all p elements
	eval := evaluator.NewEvaluator()
	results, err := eval.Evaluate(`//p`, html)
	if err != nil {
		fmt.Printf("Error evaluating: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d p elements:\n", len(results))
	for i, result := range results {
		fmt.Printf("P element %d: '%s'\n", i+1, result.TextContent)
		
		// We need to find this node in the parsed tree to check its parents
		findAndPrintAncestors(document, result.TextContent)
		fmt.Println()
	}
}

func findAndPrintAncestors(node *types.Node, targetText string) {
	if node.TextContent == targetText && node.Name == "p" {
		fmt.Printf("Found target p element: %s\n", targetText)
		current := node.Parent
		fmt.Println("Ancestors:")
		for current != nil {
			fmt.Printf("  %s (type: %d)\n", current.Name, current.Type)
			current = current.Parent
		}
		return
	}
	
	// Recursively search children
	for _, child := range node.Children {
		findAndPrintAncestors(child, targetText)
	}
}