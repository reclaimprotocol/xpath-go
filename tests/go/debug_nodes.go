package main

import (
	"fmt"
	"log"
	"strings"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func main() {
	html := `<html><body><div>Has content</div><div></div><div>   </div></body></html>`

	results, err := xpath.Query("//div", html)
	if err != nil {
		log.Fatal(err)
	}

	for i, result := range results {
		fmt.Printf("Div %d:\n", i)
		fmt.Printf("  TextContent: %q\n", result.TextContent)
		fmt.Printf("  Has children: %v\n", len(result.Children) > 0)
		fmt.Printf("  TrimSpace(TextContent): %q\n", strings.TrimSpace(result.TextContent))
		fmt.Printf("  Should match not(node()): %v\n", len(result.Children) == 0 && strings.TrimSpace(result.TextContent) == "")
		fmt.Println()
	}
}
