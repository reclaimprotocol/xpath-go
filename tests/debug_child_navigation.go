package main

import (
	"fmt"
	"log"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><head><title>Test</title><meta charset='utf-8'/></head><body><main><h1>Title</h1><p>Content</p></main></body></html>`
	
	fmt.Println("=== TESTING CHILD NAVIGATION ===")
	fmt.Println()
	
	// Test basic element selection
	result1, err := xpath.Query("//html", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("//html: %d results\n", len(result1))
	
	result2, err := xpath.Query("//head", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("//head: %d results\n", len(result2))
	
	result3, err := xpath.Query("//title", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("//title: %d results\n", len(result3))
	
	// Test child path navigation  
	result4, err := xpath.Query("//head/title", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("//head/title: %d results\n", len(result4))
	
	result5, err := xpath.Query("//head/meta", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("//head/meta: %d results\n", len(result5))
	
	// Test if the issue is with child path evaluation in predicates
	result6, err := xpath.Query("//html[head]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("//html[head]: %d results\n", len(result6))
	
	result7, err := xpath.Query("//html[title]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("//html[title]: %d results\n", len(result7))
	
	fmt.Println("\nIf child paths work in general but not in predicates, the issue is in evaluateSimpleCondition")
}