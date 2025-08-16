package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test boolean AND combination
	html := `<html><body><ul><li><span>Item 1</span><!-- comment --></li><li>Item 2</li><li><a href='#'>Item 3</a><span>Extra</span></li></ul></body></html>`
	
	fmt.Println("=== Boolean AND Test ===")
	
	// Test individual conditions
	spanResults, err := xpath.Query("//li[span]", html)
	if err != nil {
		fmt.Printf("Span error: %v\n", err)
		return
	}
	fmt.Printf("//li[span]: %d results\n", len(spanResults))
	
	notAResults, err := xpath.Query("//li[not(a)]", html)
	if err != nil {
		fmt.Printf("Not(a) error: %v\n", err)
		return
	}
	fmt.Printf("//li[not(a)]: %d results\n", len(notAResults))
	
	// Test AND combination
	andResults, err := xpath.Query("//li[span and not(a)]", html)
	if err != nil {
		fmt.Printf("AND error: %v\n", err)
		return
	}
	fmt.Printf("//li[span and not(a)]: %d results\n", len(andResults))
	
	// Test simpler AND cases
	fmt.Println("\n=== Simpler AND tests ===")
	
	// Test with attributes instead
	attrHtml := `<div id="test" class="active">content</div>`
	attrAnd, err := xpath.Query("//div[@id and @class]", attrHtml)
	if err != nil {
		fmt.Printf("Attr AND error: %v\n", err)
		return
	}
	fmt.Printf("//div[@id and @class]: %d results (should be 1)\n", len(attrAnd))
	
	// Test with simple element AND
	simpleHtml := `<div><span>text</span></div>`
	simpleAnd, err := xpath.Query("//div[span and not(a)]", simpleHtml)
	if err != nil {
		fmt.Printf("Simple AND error: %v\n", err)
		return
	}
	fmt.Printf("//div[span and not(a)]: %d results (should be 1)\n", len(simpleAnd))
}