package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// HTML with just the problematic second div
	html := `<html><body><div> </div></body></html>`
	
	fmt.Println("Testing just the space-only div")
	fmt.Println("HTML:", html)
	fmt.Println()
	
	// Test the full compound expression
	fmt.Print("//div[normalize-space(text())='' and not(*)]: ")
	results, _ := xpath.Query("//div[normalize-space(text())='' and not(*)]", html)
	fmt.Printf("%s (%d results)\n", boolToString(len(results) > 0), len(results))
	
	// Test each part separately
	fmt.Print("//div[normalize-space(text())='']: ")
	results1, _ := xpath.Query("//div[normalize-space(text())='']", html)
	fmt.Printf("%s\n", boolToString(len(results1) > 0))
	
	fmt.Print("//div[not(*)]: ")
	results2, _ := xpath.Query("//div[not(*)]", html)
	fmt.Printf("%s\n", boolToString(len(results2) > 0))
	
	fmt.Println("\nBoth parts should be true, so combined should be true")
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}