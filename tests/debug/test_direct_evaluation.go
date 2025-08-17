package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go/pkg/utils"
	"github.com/reclaimprotocol/xpath-go/internal/evaluator"
)

func main() {
	html := `<html><body><div id="empty"></div><div id="space"> </div><div id="child"><span></span></div><div id="content">Content</div></body></html>`
	
	fmt.Println("Testing direct evaluation of conditions")
	fmt.Println("HTML:", html)
	fmt.Println()
	
	// Parse HTML
	parser := utils.NewHTMLParser()
	root, err := parser.Parse(html)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
		return
	}
	
	// Find the space div (second div)
	spaceDivs := findDivById(root, "space")
	if len(spaceDivs) == 0 {
		fmt.Println("Could not find space div")
		return
	}
	spaceDiv := spaceDivs[0]
	
	fmt.Printf("Found space div: text='%s', children=%d\n", spaceDiv.TextContent, len(spaceDiv.Children))
	
	// Create evaluator
	eval := evaluator.NewEvaluator()
	
	// Test individual conditions using internal methods
	fmt.Println("\nTesting individual conditions:")
	
	condition1 := "normalize-space(text())=''"
	result1 := eval.EvaluateSimpleCondition(spaceDiv, condition1) 
	fmt.Printf("1. %s: %v\n", condition1, result1)
	
	condition2 := "not(*)"
	result2 := eval.EvaluateSimpleCondition(spaceDiv, condition2)
	fmt.Printf("2. %s: %v\n", condition2, result2)
	
	// Test combined condition
	fmt.Println("\nTesting combined condition:")
	combinedCondition := "normalize-space(text())='' and not(*)"
	resultCombined := eval.EvaluateAndExpression(combinedCondition, spaceDiv)
	fmt.Printf("3. %s: %v\n", combinedCondition, resultCombined)
	
	fmt.Println("\nExpected: all should be true")
}

func findDivById(node *types.Node, id string) []*types.Node {
	var divs []*types.Node
	
	if node.Name == "div" {
		if attr, exists := node.Attributes["id"]; exists && attr == id {
			divs = append(divs, node)
		}
	}
	
	for _, child := range node.Children {
		divs = append(divs, findDivById(child, id)...)
	}
	
	return divs
}