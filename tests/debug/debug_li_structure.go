package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test step by step to isolate the issue
	html := `<html><body><ul><li><span>Item 1</span><!-- comment --></li><li>Item 2</li><li><a href='#'>Item 3</a><span>Extra</span></li></ul></body></html>`
	
	// First, check how many li elements we find
	allLi, err := xpath.Query("//li", html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Total li elements found: %d\n", len(allLi))
	
	// Check spans
	allSpan, err := xpath.Query("//span", html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Total span elements found: %d\n", len(allSpan))
	
	// Check a elements
	allA, err := xpath.Query("//a", html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Total a elements found: %d\n", len(allA))
	
	// Test li with span
	liWithSpan, err := xpath.Query("//li[span]", html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Li elements with span: %d\n", len(liWithSpan))
	
	// Test li with not(a)
	liWithNotA, err := xpath.Query("//li[not(a)]", html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("Li elements with not(a): %d\n", len(liWithNotA))
	
	// Test the simple version without boolean logic
	fmt.Println("\n=== Testing simpler version ===")
	simpleHtml := `<li><span>Item 1</span></li>`
	
	simpleLi, err := xpath.Query("//li", simpleHtml) 
	if err != nil {
		fmt.Printf("Simple li error: %v\n", err)
		return
	}
	fmt.Printf("Simple li count: %d\n", len(simpleLi))
	
	simpleSpan, err := xpath.Query("//li[span]", simpleHtml)
	if err != nil {
		fmt.Printf("Simple span error: %v\n", err)
		return
	}
	fmt.Printf("Simple li[span] count: %d\n", len(simpleSpan))
}