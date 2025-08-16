package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test span detection specifically
	html := `<ul><li>Item 2</li></ul>`

	fmt.Println("Testing span detection on: <ul><li>Item 2</li></ul>")
	fmt.Println("Expected: NO spans found")
	fmt.Println()

	// Find all spans in document
	allSpans, err := xpath.Query("//span", html)
	if err != nil {
		fmt.Printf("Find all spans ERROR: %v\n", err)
		return
	}
	fmt.Printf("All spans in document: %d\n", len(allSpans))

	// Find all li elements
	allLi, err := xpath.Query("//li", html)
	if err != nil {
		fmt.Printf("Find all li ERROR: %v\n", err)
		return
	}
	fmt.Printf("All li elements: %d\n", len(allLi))

	// Test li[span] - this should return 0
	liWithSpan, err := xpath.Query("//li[span]", html)
	if err != nil {
		fmt.Printf("Li with span ERROR: %v\n", err)
		return
	}
	fmt.Printf("Li elements with span children: %d (should be 0)\n", len(liWithSpan))

	// Test if the issue is with child vs descendant
	liWithSpanChild, err := xpath.Query("//li[child::span]", html)
	if err != nil {
		fmt.Printf("Li with span child ERROR: %v\n", err)
		return
	}
	fmt.Printf("Li elements with span via child axis: %d (should be 0)\n", len(liWithSpanChild))

	// Test what children the li actually has
	liChildren, err := xpath.Query("//li/node()", html)
	if err != nil {
		fmt.Printf("Li children ERROR: %v\n", err)
		return
	}
	fmt.Printf("Actual li children: %d\n", len(liChildren))
	for i, child := range liChildren {
		fmt.Printf("  Child %d: %s\n", i+1, child.NodeName)
	}
}
