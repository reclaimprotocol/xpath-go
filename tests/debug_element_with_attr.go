package main

import (
	"fmt"
	"log"

	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><body><div class='widget'><span class='loading' style='display:none'>Loading...</span><div class='content'>Loaded Content</div></div></body></html>`
	
	fmt.Println("=== TESTING ELEMENT WITH ATTRIBUTE EXPRESSIONS ===")
	fmt.Println()
	
	// Test element with attribute as standalone
	result1, err := xpath.Query("//span[@class='loading']", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("//span[@class='loading']: %d results\n", len(result1))
	
	// Test if evaluateSimpleCondition can handle element with attributes
	// by testing it within a parent context where it would be called by evaluateSimpleCondition
	result2, err := xpath.Query("//div[span[@class='loading']]", htmlContent) 
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("//div[span[@class='loading']]: %d results\n", len(result2))
	
	result3, err := xpath.Query("//div[div[@class='content']]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("//div[div[@class='content']]: %d results\n", len(result3))
	
	fmt.Println("\nIf these work individually but fail in AND, the issue is in complex boolean evaluation")
}