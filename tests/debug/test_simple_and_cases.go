package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div id="test" class="active">Content</div></body></html>`
	
	fmt.Println("Testing different types of and expressions")
	fmt.Println("HTML:", html)
	fmt.Println()
	
	// Test simple boolean without functions (should route to SimpleBooleanType)
	fmt.Println("1. Simple boolean (no functions):")
	fmt.Print("   //div[@id='test' and @class='active']: ")
	results1, _ := xpath.Query("//div[@id='test' and @class='active']", html)
	fmt.Printf("%s\n", boolToString(len(results1) > 0))
	
	// Test boolean with one function (should route to ComplexBooleanType)
	fmt.Println("\n2. Boolean with text() function:")
	fmt.Print("   //div[@id='test' and text()='Content']: ")
	results2, _ := xpath.Query("//div[@id='test' and text()='Content']", html)
	fmt.Printf("%s\n", boolToString(len(results2) > 0))
	
	// Test our problematic case
	html2 := `<html><body><div id="empty"></div><div id="space"> </div></body></html>`
	fmt.Println("\n3. Our problematic case:")
	fmt.Printf("HTML: %s\n", html2)
	fmt.Print("   //div[@id='space'][normalize-space(text())='' and not(*)]: ")
	results3, _ := xpath.Query("//div[@id='space'][normalize-space(text())='' and not(*)]", html2)
	fmt.Printf("%s\n", boolToString(len(results3) > 0))
	
	fmt.Println("\nThis will help us understand if the issue is with ComplexBooleanType evaluation")
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}