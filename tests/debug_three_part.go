package main

import (
	"fmt"
	"log"

	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><body><div class='primary inactive'>Item 3</div></body></html>`

	fmt.Println("=== TESTING THREE-PART AND LOGIC ===")
	fmt.Println()

	// Test the exact failing expression
	result, err := xpath.Query("//div[contains(@class, 'primary') and contains(@class, 'active') and not(contains(@class, 'inactive'))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Full 3-part: %d results\n", len(result))

	// Expected logic:
	// Split 1: contains(@class, 'primary') AND [contains(@class, 'active') and not(contains(@class, 'inactive'))]
	// = 1 AND 0 = 0

	left, err := xpath.Query("//div[contains(@class, 'primary')]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}

	right, err := xpath.Query("//div[contains(@class, 'active') and not(contains(@class, 'inactive'))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Expected: LEFT(%d) AND RIGHT(%d) = %d\n", len(left), len(right), 0)
	fmt.Printf("Actual: %d\n", len(result))

	if len(result) != 0 {
		fmt.Println("MISMATCH: Complex boolean evaluator is not properly handling the final AND")
	} else {
		fmt.Println("SUCCESS: Three-part AND logic is working correctly!")
	}
}
