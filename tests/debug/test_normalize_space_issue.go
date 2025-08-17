package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div id="space"> </div></body></html>`
	
	fmt.Println("Testing normalize-space issue specifically")
	fmt.Println("HTML:", html)
	fmt.Println()
	
	// Test normalize-space alone
	fmt.Print("//div[normalize-space(text())='']: ")
	results1, _ := xpath.Query("//div[normalize-space(text())='']", html)
	fmt.Printf("%s (%d results)\n", boolToString(len(results1) > 0), len(results1))
	
	// Test what normalize-space actually returns for space
	fmt.Print("//div[@id='space'][normalize-space(text())='']: ")
	results2, _ := xpath.Query("//div[@id='space'][normalize-space(text())='']", html)
	fmt.Printf("%s\n", boolToString(len(results2) > 0))
	
	fmt.Print("//div[@id='space'][normalize-space(text())=' ']: ")
	results3, _ := xpath.Query("//div[@id='space'][normalize-space(text())=' ']", html)
	fmt.Printf("%s\n", boolToString(len(results3) > 0))
	
	fmt.Println("\nExpected: normalize-space(' ') should equal ''")
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}