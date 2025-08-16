package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><ul><li><span>Item 1</span><!-- comment --></li><li>Item 2</li><li><a href='#'>Item 3</a><span>Extra</span></li></ul></body></html>`

	fmt.Println("Testing: //li[span and not(a)]")
	fmt.Println("Expected: 1 result (first li)")
	fmt.Println()

	// Test all li elements
	allLi, err := xpath.Query("//li", html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	fmt.Printf("Found %d li elements\n", len(allLi))

	// Test each li individually
	for i := 1; i <= len(allLi); i++ {
		// Test span existence
		spanResults, err := xpath.Query(fmt.Sprintf("(//li)[%d][span]", i), html)
		if err != nil {
			fmt.Printf("Li %d span test ERROR: %v\n", i, err)
		} else {
			fmt.Printf("Li %d has span: %v\n", i, len(spanResults) > 0)
		}

		// Test not(a)
		notAResults, err := xpath.Query(fmt.Sprintf("(//li)[%d][not(a)]", i), html)
		if err != nil {
			fmt.Printf("Li %d not(a) test ERROR: %v\n", i, err)
		} else {
			fmt.Printf("Li %d matches not(a): %v\n", i, len(notAResults) > 0)
		}

		// Test combined
		combinedResults, err := xpath.Query(fmt.Sprintf("(//li)[%d][span and not(a)]", i), html)
		if err != nil {
			fmt.Printf("Li %d combined test ERROR: %v\n", i, err)
		} else {
			fmt.Printf("Li %d matches combined: %v\n", i, len(combinedResults) > 0)
		}
		fmt.Println()
	}

	// Test all together
	finalResults, err := xpath.Query("//li[span and not(a)]", html)
	if err != nil {
		fmt.Printf("Final test ERROR: %v\n", err)
	} else {
		fmt.Printf("Final result: %d (expected 1)\n", len(finalResults))
	}
}
