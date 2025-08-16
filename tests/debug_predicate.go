package main

import (
	"fmt"
	"strings"

	"github.com/reclaimprotocol/xpath-go/internal/evaluator"
	"github.com/reclaimprotocol/xpath-go/pkg/types"
	"github.com/reclaimprotocol/xpath-go/pkg/utils"
)

func main() {
	html := `<html><body><div class="red">A</div><div class="blue">B</div></body></html>`

	// Parse HTML
	htmlParser := utils.NewHTMLParser()
	document, err := htmlParser.Parse(html)
	if err != nil {
		fmt.Printf("HTML parse error: %v\n", err)
		return
	}

	// Create a test div node with class="red"
	testNode := &types.Node{
		Type:        types.ElementNode,
		Name:        "div",
		Attributes:  map[string]string{"class": "red"},
		TextContent: "A",
	}

	// Test the evaluateSimpleCondition function directly
	eval := evaluator.NewEvaluator()

	fmt.Println("=== Testing evaluateSimpleCondition directly ===")

	// This should work
	result1 := eval.EvaluateSimpleCondition(testNode, "@class='red'")
	fmt.Printf("@class='red': %v\n", result1)

	result2 := eval.EvaluateSimpleCondition(testNode, "@class='blue'")
	fmt.Printf("@class='blue': %v\n", result2)
}
