package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <html_file>\n", os.Args[0])
		os.Exit(1)
	}

	htmlFile := os.Args[1]
	htmlContent, err := os.ReadFile(htmlFile)
	if err != nil {
		fmt.Printf("Failed to read HTML file: %v\n", err)
		return
	}

	html := string(htmlContent)

	// Test individual parts
	queries := []string{
		"//div",
		"//div[normalize-space(text())='']",
		"//div[not(*)]",
		"//div[normalize-space(text())='' and not(*)]",
	}

	for _, query := range queries {
		results, err := xpath.Query(query, html)
		if err != nil {
			fmt.Printf("Query: %s - ERROR: %v\n", query, err)
		} else {
			fmt.Printf("Query: %s - Results: %d\n", query, len(results))
			for i, result := range results {
				fmt.Printf("  %d: text=\"%s\" children=%d\n", i+1, result.TextContent, countChildren(result.Value))
			}
		}
		fmt.Println()
	}
}

func countChildren(value string) int {
	// Simple heuristic to count element children by counting opening tags
	// This is just for debugging, not production-ready
	count := 0
	for i := 0; i < len(value)-1; i++ {
		if value[i] == '<' && value[i+1] != '/' && value[i+1] != '!' {
			count++
		}
	}
	return count
}
