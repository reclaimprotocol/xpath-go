package main

import (
	"fmt"
	"log"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><body><div class='widget'><span class='loading' style='display:none'>Loading...</span><div class='content'>Loaded Content</div></div></body></html>`

	fmt.Println("=== TESTING evaluateSimpleCondition LOGIC ===")
	fmt.Println()

	// Get the widget div first
	widgetDivs, err := xpath.Query("//div[@class='widget']", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Found %d widget divs\n", len(widgetDivs))

	// Test what should work when evaluateSimpleCondition is called
	// from the context of the widget div

	// These should work (and do work when used directly in predicates)
	result1, err := xpath.Query("//div[@class='widget'][span[@class='loading']]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Direct predicate [span[@class='loading']]: %d\n", len(result1))

	result2, err := xpath.Query("//div[@class='widget'][div[@class='content']]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Direct predicate [div[@class='content']]: %d\n", len(result2))

	// Test if the issue is specific to the complex boolean context
	// by testing a working case vs failing case
	result3, err := xpath.Query("//div[@class='widget'][@class='widget' and @class='widget']", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Attribute AND (should work): %d\n", len(result3))

	fmt.Println("\nThe issue is likely that complex boolean evaluator")
	fmt.Println("calls evaluateSimpleCondition with nested element expressions")
	fmt.Println("but evaluateSimpleCondition may not handle them correctly in that context")
}
