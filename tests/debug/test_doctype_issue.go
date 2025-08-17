package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	fmt.Println("🔍 Testing DOCTYPE parsing issue...")
	
	// Test 1: HTML with DOCTYPE - this should fail according to the issue
	fmt.Println("\n1. Testing HTML with DOCTYPE:")
	htmlWithDoctype := `<!DOCTYPE html><html><head><title>Test</title></head></html>`
	fmt.Printf("HTML: %s\n", htmlWithDoctype)
	
	matches1, err1 := xpath.Query("//title", htmlWithDoctype)
	if err1 != nil {
		fmt.Printf("❌ ERROR: %v\n", err1)
	} else {
		fmt.Printf("✅ SUCCESS: Found %d matches\n", len(matches1))
		for i, match := range matches1 {
			fmt.Printf("  %d. %s: '%s'\n", i+1, match.NodeName, match.TextContent)
		}
	}
	
	// Test 2: HTML without DOCTYPE - this should work
	fmt.Println("\n2. Testing HTML without DOCTYPE:")
	htmlWithoutDoctype := `<html><head><title>Test</title></head></html>`
	fmt.Printf("HTML: %s\n", htmlWithoutDoctype)
	
	matches2, err2 := xpath.Query("//title", htmlWithoutDoctype)
	if err2 != nil {
		fmt.Printf("❌ ERROR: %v\n", err2)
	} else {
		fmt.Printf("✅ SUCCESS: Found %d matches\n", len(matches2))
		for i, match := range matches2 {
			fmt.Printf("  %d. %s: '%s'\n", i+1, match.NodeName, match.TextContent)
		}
	}
	
	// Test 3: More complex DOCTYPE
	fmt.Println("\n3. Testing complex DOCTYPE:")
	htmlComplexDoctype := `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd"><html><head><title>Complex</title></head></html>`
	fmt.Printf("HTML: %s\n", htmlComplexDoctype)
	
	matches3, err3 := xpath.Query("//title", htmlComplexDoctype)
	if err3 != nil {
		fmt.Printf("❌ ERROR: %v\n", err3)
	} else {
		fmt.Printf("✅ SUCCESS: Found %d matches\n", len(matches3))
		for i, match := range matches3 {
			fmt.Printf("  %d. %s: '%s'\n", i+1, match.NodeName, match.TextContent)
		}
	}
}