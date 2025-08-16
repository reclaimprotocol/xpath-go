package main

import (
	"fmt"
	"log"

	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><body><div class='primary inactive'>Item 3</div></body></html>`

	fmt.Println("=== TESTING NOT() IN AND EXPRESSIONS ===")
	fmt.Println()

	// Test individual conditions
	fmt.Println("Testing individual conditions:")

	result1, err := xpath.Query("//div[contains(@class, 'primary')]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("contains(@class, 'primary'): %d results\n", len(result1))

	result2, err := xpath.Query("//div[contains(@class, 'active')]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("contains(@class, 'active'): %d results\n", len(result2))

	result3, err := xpath.Query("//div[contains(@class, 'inactive')]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("contains(@class, 'inactive'): %d results\n", len(result3))

	result4, err := xpath.Query("//div[not(contains(@class, 'inactive'))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("not(contains(@class, 'inactive')): %d results\n", len(result4))

	fmt.Println("\nTesting combinations:")

	// Test two-part AND
	result5, err := xpath.Query("//div[contains(@class, 'primary') and contains(@class, 'active')]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("primary AND active: %d results\n", len(result5))

	result6, err := xpath.Query("//div[contains(@class, 'primary') and not(contains(@class, 'inactive'))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("primary AND not(inactive): %d results\n", len(result6))

	result7, err := xpath.Query("//div[contains(@class, 'active') and not(contains(@class, 'inactive'))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("active AND not(inactive): %d results\n", len(result7))

	// Test full three-part AND
	result8, err := xpath.Query("//div[contains(@class, 'primary') and contains(@class, 'active') and not(contains(@class, 'inactive'))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("primary AND active AND not(inactive): %d results\n", len(result8))
}
