package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test if span is being parsed correctly
	html := `<html><body><div><span></span></div></body></html>`
	
	// Check if we can find the span
	spanResults, err := xpath.Query("//span", html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	
	fmt.Printf("Found %d spans\n", len(spanResults))
	
	// Check if the div has a span child
	divWithSpanResults, err := xpath.Query("//div[span]", html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	
	fmt.Printf("Found %d divs with span children: %v\n", len(divWithSpanResults), len(divWithSpanResults) > 0)
	
	// Check the div itself
	divResults, err := xpath.Query("//div", html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	
	fmt.Printf("Found %d divs\n", len(divResults))
	
	// Test not(*) on this div
	notResults, err := xpath.Query("//div[not(*)]", html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	
	fmt.Printf("Divs matching not(*): %d (should be 0)\n", len(notResults))
}