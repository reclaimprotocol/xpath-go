package main

import (
	"fmt"
	"log"

	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	htmlContent := `<html><body><div class='widget'><span class='loading' style='display:none'>Loading...</span><div class='content'>Loaded Content</div></div></body></html>`

	fmt.Println("=== TESTING NESTED ELEMENT CHECKING ===")
	fmt.Println()

	// Test individual parts
	fmt.Println("Testing individual conditions:")

	result1, err := xpath.Query("//div[@class='widget']", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("div[@class='widget']: %d results\n", len(result1))

	result2, err := xpath.Query("//span[@class='loading']", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("span[@class='loading']: %d results\n", len(result2))

	result3, err := xpath.Query("//div[@class='content']", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("div[@class='content']: %d results\n", len(result3))

	// Test element existence within widget div
	fmt.Println("\nTesting element existence within widget div:")

	result4, err := xpath.Query("//div[@class='widget'][span]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("widget div with any span: %d results\n", len(result4))

	result5, err := xpath.Query("//div[@class='widget'][div]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("widget div with any div: %d results\n", len(result5))

	result6, err := xpath.Query("//div[@class='widget'][span[@class='loading']]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("widget div with loading span: %d results\n", len(result6))

	result7, err := xpath.Query("//div[@class='widget'][div[@class='content']]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("widget div with content div: %d results\n", len(result7))

	// Test the AND combination
	fmt.Println("\nTesting AND combination:")

	result8, err := xpath.Query("//div[@class='widget'][span[@class='loading'] and div[@class='content']]", htmlContent)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("full expression: %d results\n", len(result8))

	fmt.Println("\nExpected: The widget div contains both elements, so should return 1 result")
}
