package main

import (
	"fmt"
	"log"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><head><title>Test</title><meta charset='utf-8'/></head><body><main><h1>Title</h1><p>Content</p></main></body></html>`
	
	fmt.Println("=== TESTING CHILD PATH PREDICATES ===")
	fmt.Println()
	
	// Test step by step what should work
	result1, err := xpath.Query("//html[head/title]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("//html[head/title]: %d results\n", len(result1))
	
	result2, err := xpath.Query("//html[head/meta]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("//html[head/meta]: %d results\n", len(result2))
	
	result3, err := xpath.Query("//html[body/main]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("//html[body/main]: %d results\n", len(result3))
	
	// Test the failing case specifically
	result4, err := xpath.Query("/html[head/title]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("/html[head/title]: %d results\n", len(result4))
	
	fmt.Println("\nIf these return 1, the child path logic is working")
}