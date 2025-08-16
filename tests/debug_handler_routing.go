package main

import (
	"fmt"
	"log"

	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><body><div class='primary inactive'>Item 3</div></body></html>`

	fmt.Println("=== TESTING HANDLER ROUTING ===")
	fmt.Println()

	// Test expressions that should go to different handlers

	// This should go to simple AND handler (2 parts, no functions)
	fmt.Println("Testing simple AND (should route to applyAndPredicate):")
	result1, err := xpath.Query("//div[@class and @id]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("@class and @id: %d results\n", len(result1))

	// This should go to complex boolean handler (has functions)
	fmt.Println("\nTesting complex boolean (should route to applyComplexBooleanPredicate):")
	result2, err := xpath.Query("//div[contains(@class, 'primary') and contains(@class, 'inactive')]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("contains() and contains(): %d results\n", len(result2))

	// This should also go to complex boolean handler (has functions + not)
	result3, err := xpath.Query("//div[contains(@class, 'primary') and not(contains(@class, 'inactive'))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("contains() and not(contains()): %d results\n", len(result3))

	// Test with parentheses to force complex boolean
	result4, err := xpath.Query("//div[(contains(@class, 'primary')) and (not(contains(@class, 'inactive')))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("(contains()) and (not(contains())): %d results\n", len(result4))

	fmt.Println("\nIf results differ, it indicates different handlers are being used")
}
