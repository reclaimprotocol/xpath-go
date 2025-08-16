package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Use the exact HTML from our test file
	html := `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`

	// Test span finding
	spanResults, err := xpath.Query("//span", html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	fmt.Printf("Found %d spans\n", len(spanResults))

	// Test div with span
	divWithSpanResults, err := xpath.Query("//div[span]", html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	fmt.Printf("Found %d divs with span children\n", len(divWithSpanResults))

	// Test not(*) on each div position
	for i := 1; i <= 4; i++ {
		notResults, err := xpath.Query(fmt.Sprintf("(//div)[%d][not(*)]", i), html)
		if err != nil {
			fmt.Printf("ERROR testing div %d: %v\n", i, err)
		} else {
			fmt.Printf("Div %d matches not(*): %v\n", i, len(notResults) > 0)
		}
	}

	// Test all divs with not(*)
	allNotResults, err := xpath.Query("//div[not(*)]", html)
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	fmt.Printf("Total divs matching not(*): %d\n", len(allNotResults))
}
