package main

import (
	"fmt"
	"log"

	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><body><div class='primary inactive'>Item 3</div></body></html>`

	fmt.Println("=== BOOLEAN LOGIC TRACING ===")
	fmt.Println()

	// Test with simple AND that should pass through complex boolean logic
	result1, err := xpath.Query("//div[contains(@class, 'primary') and contains(@class, 'inactive')]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("primary AND inactive: %d results (should be 1)\n", len(result1))

	// Test with simple AND that should fail
	result2, err := xpath.Query("//div[contains(@class, 'primary') and contains(@class, 'nonexistent')]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("primary AND nonexistent: %d results (should be 0)\n", len(result2))

	// Test with NOT that should pass
	result3, err := xpath.Query("//div[contains(@class, 'primary') and not(contains(@class, 'nonexistent'))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("primary AND not(nonexistent): %d results (should be 1)\n", len(result3))

	// Test with NOT that should fail
	result4, err := xpath.Query("//div[contains(@class, 'primary') and not(contains(@class, 'inactive'))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("primary AND not(inactive): %d results (should be 0)\n", len(result4))

	fmt.Println("\nPattern analysis:")
	fmt.Println("If all AND results are inverted, there's a bug in the complex boolean evaluator")
}
