package main

import (
	"fmt"
	"log"

	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><body><div class='primary inactive'>Item 3</div></body></html>`
	
	fmt.Println("=== TESTING not() IN evaluateSimpleCondition ===")
	fmt.Println()
	
	// Test the exact condition that should be evaluated
	fmt.Println("Individual tests:")
	
	// Test contains() by itself
	result1, err := xpath.Query("//div[contains(@class, 'inactive')]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("contains(@class, 'inactive'): %d results\n", len(result1))
	
	// Test not(contains()) by itself
	result2, err := xpath.Query("//div[not(contains(@class, 'inactive'))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("not(contains(@class, 'inactive')): %d results\n", len(result2))
	
	// Test a simple 2-part AND that should work
	result3, err := xpath.Query("//div[contains(@class, 'primary') and not(contains(@class, 'inactive'))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("primary AND not(inactive): %d results\n", len(result3))
	
	// Test with a different attribute to isolate the issue
	result4, err := xpath.Query("//div[contains(@class, 'primary') and not(contains(@class, 'nonexistent'))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("primary AND not(nonexistent): %d results\n", len(result4))
	
	fmt.Println("\nExpected results:")
	fmt.Println("contains(@class, 'inactive'): 1 (because 'primary inactive' contains 'inactive')")
	fmt.Println("not(contains(@class, 'inactive')): 0 (inverse of above)")
	fmt.Println("primary AND not(inactive): 0 (1 AND 0 = 0)")
	fmt.Println("primary AND not(nonexistent): 1 (1 AND 1 = 1)")
}