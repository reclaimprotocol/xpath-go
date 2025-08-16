package main

import (
	"fmt"
	"log"

	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><body><div class='widget'><span class='loading' style='display:none'>Loading...</span><div class='content'>Loaded Content</div></div></body></html>`
	
	fmt.Println("=== TESTING ELEMENT AND LOGIC ===")
	fmt.Println()
	
	// Test simple element AND (should work if our fix was complete)
	result1, err := xpath.Query("//div[@class='widget'][span and div]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("widget div with span AND div: %d results\n", len(result1))
	
	// Test attribute AND (should work from before)
	result2, err := xpath.Query("//span[@class and @style]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("span with class AND style: %d results\n", len(result2))
	
	// Test mixed attribute element AND
	result3, err := xpath.Query("//div[@class='widget'][span and @class]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("widget div with span AND @class: %d results\n", len(result3))
	
	// Test the specific failing case again  
	result4, err := xpath.Query("//div[@class='widget'][span[@class='loading'] and div[@class='content']]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("specific failing case: %d results\n", len(result4))
	
	fmt.Println("\nIf simple element AND works but complex doesn't, it's a routing issue")
	fmt.Println("If simple element AND also fails, it's a broader element AND logic issue")
}