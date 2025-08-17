package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><span class='item'>Item 1</span><span class='item'>Item 2</span><span class='item'>Item 3</span><span class='item'>Item 4</span></body></html>`
	xpath1 := `//span[@class='item'][position() mod 2 = 0]`
	
	fmt.Println("Testing Position Mod 2")
	fmt.Println("HTML:", html)
	fmt.Println("XPath:", xpath1)
	fmt.Println()
	
	results, err := xpath.Query(xpath1, html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Found %d results:\n", len(results))
	for i, r := range results {
		fmt.Printf("%d. NodeName: %s, Text: '%s'\n", i+1, r.NodeName, r.TextContent)
		fmt.Printf("   Path: %s\n", r.Path)
	}
	
	fmt.Println("\n--- Expected Results ---")
	fmt.Println("JavaScript finds 2 nodes:")
	fmt.Println("1. Item 2 (position 2, 2 mod 2 = 0)")
	fmt.Println("2. Item 4 (position 4, 4 mod 2 = 0)")
	fmt.Println("Should select even-positioned spans with class='item'")
	
	// Test components separately
	fmt.Println("\n--- Debug Components ---")
	
	// Test class filter only
	results1, _ := xpath.Query("//span[@class='item']", html)
	fmt.Printf("//span[@class='item'] finds %d results:\n", len(results1))
	for i, r := range results1 {
		fmt.Printf("  %d. '%s'\n", i+1, r.TextContent)
	}
	
	// Test position() only (should fail since mod not implemented)
	results2, _ := xpath.Query("//span[position() mod 2 = 0]", html)
	fmt.Printf("//span[position() mod 2 = 0] finds %d results:\n", len(results2))
	for i, r := range results2 {
		fmt.Printf("  %d. '%s'\n", i+1, r.TextContent)
	}
}