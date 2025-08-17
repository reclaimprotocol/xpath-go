package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Create a simpler test case to isolate the and logic
	html := `<html><body><div id="empty"></div><div id="space"> </div><div id="child"><span></span></div><div id="content">Content</div></body></html>`
	
	fmt.Println("Testing and logic without position() predicates")
	fmt.Println("HTML:", html)
	fmt.Println()
	
	// Test the second div specifically using its id
	fmt.Println("Testing div with id='space' (contains single space):")
	
	// Individual conditions
	fmt.Print("1. //div[@id='space'][normalize-space(text())='']: ")
	results1, _ := xpath.Query("//div[@id='space'][normalize-space(text())='']", html)
	fmt.Printf("%s\n", boolToString(len(results1) > 0))
	
	fmt.Print("2. //div[@id='space'][not(*)]: ")
	results2, _ := xpath.Query("//div[@id='space'][not(*)]", html)
	fmt.Printf("%s\n", boolToString(len(results2) > 0))
	
	// Combined condition
	fmt.Print("3. //div[@id='space'][normalize-space(text())='' and not(*)]: ")
	results3, _ := xpath.Query("//div[@id='space'][normalize-space(text())='' and not(*)]", html)
	fmt.Printf("%s\n", boolToString(len(results3) > 0))
	
	fmt.Println()
	fmt.Println("Expected: All three should be true")
}

func boolToString(b bool) string {
	if b {
		return "true"  
	}
	return "false"
}