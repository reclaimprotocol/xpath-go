package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go/internal/parser"
)

func main() {
	p := parser.NewParser()

	fmt.Println("Testing tokenization:")

	testCases := []string{
		"//div",
		"/html",
		"div",
	}

	for _, xpath := range testCases {
		fmt.Printf("\n--- Tokenizing: %s ---\n", xpath)

		// We need to access the private tokenize method
		// For now, let's just check what characters we have
		fmt.Printf("Characters: ")
		for i, char := range xpath {
			fmt.Printf("[%d]='%c' ", i, char)
		}
		fmt.Println()

		// Check if 'd' in "div" would pass isNameStart
		for i, char := range xpath {
			if char == 'd' {
				fmt.Printf("Character 'd' at position %d: isLetter=%v, isNameStart=%v\n",
					i, isLetter(byte(char)), isNameStart(byte(char)))
			}
		}
	}
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isNameStart(c byte) bool {
	return isLetter(c) || c == '_'
}
