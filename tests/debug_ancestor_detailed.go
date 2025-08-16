package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/reclaimprotocol/xpath-go/internal/evaluator"
	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Debug version of hasAncestor
func hasAncestorDebug(node *types.Node, ancestorType string) bool {
	fmt.Printf("Checking ancestors of node: %s (text: %s)\n", node.Name, node.TextContent)
	current := node.Parent
	for current != nil {
		fmt.Printf("  Ancestor: %s (type: %d)\n", current.Name, current.Type)
		if current.Type == types.ElementNode && strings.ToLower(current.Name) == strings.ToLower(ancestorType) {
			fmt.Printf("  Found matching ancestor: %s\n", ancestorType)
			return true
		}
		current = current.Parent
	}
	fmt.Printf("  No matching ancestor found for: %s\n", ancestorType)
	return false
}

func main() {
	html := `<html><body><article><header><h1>Title</h1></header><section><p>Content</p><aside><p>Sidebar</p></aside></section></article></body></html>`
	xpath := `//article//section//p`

	eval := evaluator.NewEvaluator()
	results, err := eval.Evaluate(xpath, html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("All p elements in section: %d\n", len(results))
	for i, result := range results {
		fmt.Printf("Result %d: %s\n", i+1, result.TextContent)

		// Check ancestor manually
		resultNode := &types.Node{
			Name:        result.NodeName,
			TextContent: result.TextContent,
			Type:        types.NodeType(result.NodeType),
			Parent:      nil, // We need to reconstruct this...
		}
		// This won't work because we've lost the parent relationships
	}
}
