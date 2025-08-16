package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test just the first div to see what happens
	html := `<html><body><div></div></body></html>`
	query := `//div[normalize-space(text())='' and not(*)]`

	results, err := xpath.Query(query, html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	fmt.Printf("Simple test - Results: %d\n", len(results))

	// Test with second div
	html2 := `<html><body><div> </div></body></html>`
	results2, err := xpath.Query(query, html2)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	fmt.Printf("Whitespace test - Results: %d\n", len(results2))

	// Test both together
	html3 := `<html><body><div></div><div> </div></body></html>`
	results3, err := xpath.Query(query, html3)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	fmt.Printf("Both test - Results: %d\n", len(results3))
}
