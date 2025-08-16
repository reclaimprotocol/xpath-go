package main

import (
	"fmt"
	"log"

	xpath "github.com/reclaimprotocol/xpath-go"
	"github.com/reclaimprotocol/xpath-go/internal/evaluator"
)

func main() {
	htmlContent := `<html><body><div class='widget'><span class='loading' style='display:none'>Loading...</span><div class='content'>Loaded Content</div></div></body></html>`

	fmt.Println("=== DETAILED TEST 68 DEBUG ===")
	fmt.Println()

	// Test the exact failing expression
	fullExpr := "span[@class='loading'] and div[@class='content']"
	fmt.Printf("Expression: %s\n", fullExpr)

	// Check how the router classifies this
	predicateType, metadata := evaluator.ClassifyPredicate(fullExpr)
	fmt.Printf("Router Classification: Type %d, Metadata: %+v\n", predicateType, metadata)

	// Test the full XPath
	result, err := xpath.Query("//div[@class='widget'][span[@class='loading'] and div[@class='content']]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Full XPath result: %d\n", len(result))

	// Test individual parts to isolate the issue
	result1, err := xpath.Query("//div[@class='widget'][span[@class='loading']]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Left side only: %d\n", len(result1))

	result2, err := xpath.Query("//div[@class='widget'][div[@class='content']]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Right side only: %d\n", len(result2))

	// Test simpler boolean to see if it's a general issue
	result3, err := xpath.Query("//div[@class='widget'][span and div]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Simple boolean (span and div): %d\n", len(result3))

	fmt.Println("\nExpected: All should return 1 result")
}
