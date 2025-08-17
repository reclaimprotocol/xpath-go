package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div id="space"> </div></body></html>`
	
	fmt.Println("Testing single predicate with and")
	fmt.Println("HTML:", html)
	fmt.Println()
	
	// Test with just the and predicate (no id filtering)
	fmt.Print("//div[normalize-space(text())='' and not(*)]: ")
	results, _ := xpath.Query("//div[normalize-space(text())='' and not(*)]", html)
	fmt.Printf("%s\n", boolToString(len(results) > 0))
	
	fmt.Println("\nThis should show debug output if RoutePredicateExpression is called")
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}