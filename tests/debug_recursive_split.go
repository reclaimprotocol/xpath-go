package main

import (
	"fmt"
	"log"

	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><body><div class='primary inactive'>Item 3</div></body></html>`
	
	fmt.Println("=== TESTING RECURSIVE SPLIT LOGIC ===")
	fmt.Println()
	
	// Test the parts that should result from the recursive split
	fmt.Println("Testing recursive evaluation parts:")
	
	// First split: contains(@class, 'primary') AND [contains(@class, 'active') and not(contains(@class, 'inactive'))]
	left1, err := xpath.Query("//div[contains(@class, 'primary')]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("LEFT1: contains(@class, 'primary'): %d results\n", len(left1))
	
	// This is what should be evaluated recursively as the right side of first split
	right1, err := xpath.Query("//div[contains(@class, 'active') and not(contains(@class, 'inactive'))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("RIGHT1: contains(@class, 'active') and not(contains(@class, 'inactive')): %d results\n", len(right1))
	
	// Second split within RIGHT1: contains(@class, 'active') AND not(contains(@class, 'inactive'))
	left2, err := xpath.Query("//div[contains(@class, 'active')]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("LEFT2: contains(@class, 'active'): %d results\n", len(left2))
	
	right2, err := xpath.Query("//div[not(contains(@class, 'inactive'))]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("RIGHT2: not(contains(@class, 'inactive')): %d results\n", len(right2))
	
	fmt.Println("\nExpected logic:")
	fmt.Println("LEFT2 (1) AND RIGHT2 (0) should equal RIGHT1 (0)")
	fmt.Printf("But RIGHT1 is actually %d\n", len(right1))
	fmt.Println("This means the recursive evaluation in complex boolean logic is wrong")
}