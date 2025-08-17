package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div> </div></body></html>`
	
	fmt.Println("Testing space-only div with individual conditions:")
	
	// Test the normalize-space condition that should pass
	fmt.Print("1. normalize-space(text())='': ")
	results1, _ := xpath.Query("//div[normalize-space(text())='']", html)
	fmt.Printf("%s\n", boolToString(len(results1) > 0))
	
	// Test the not(*) condition that should pass
	fmt.Print("2. not(*): ")
	results2, _ := xpath.Query("//div[not(*)]", html)
	fmt.Printf("%s\n", boolToString(len(results2) > 0))
	
	// Test the compound condition that fails
	fmt.Print("3. Combined: ")
	results3, _ := xpath.Query("//div[normalize-space(text())='' and not(*)]", html)
	fmt.Printf("%s\n", boolToString(len(results3) > 0))
	
	fmt.Println("\nAll three should be true for a space-only div")
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}