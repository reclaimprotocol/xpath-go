package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test the simplest case piece by piece
	html := `<html><body><div></div></body></html>`

	queries := []string{
		"//div",
		"//div[normalize-space(text())='']",
		"//div[not(*)]",
	}

	for _, query := range queries {
		results, err := xpath.Query(query, html)
		if err != nil {
			fmt.Printf("Query: %s - ERROR: %v\n", query, err)
		} else {
			fmt.Printf("Query: %s - Results: %d\n", query, len(results))
		}
	}
}
