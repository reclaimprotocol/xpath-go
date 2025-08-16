package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Simulate the exact substring logic from our implementation
func debugSubstringExtraction(sourceText string, startExpr string) (string, string) {
	fmt.Printf("\n=== DEBUGGING SUBSTRING EXTRACTION ===\n")
	fmt.Printf("Source text: '%s' (length: %d)\n", sourceText, len(sourceText))
	fmt.Printf("Start expression: '%s'\n", startExpr)

	// Parse start position (like our implementation)
	startPos := 1
	if strings.Contains(startExpr, "string-length(text())") {
		textLength := len(sourceText)
		fmt.Printf("Text length: %d\n", textLength)

		if strings.Contains(startExpr, " - ") {
			parts := strings.Split(startExpr, " - ")
			fmt.Printf("Split parts: %v\n", parts)
			if len(parts) == 2 {
				if offset, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					startPos = textLength - offset
					fmt.Printf("Calculated start position: %d - %d = %d\n", textLength, offset, startPos)
				} else {
					fmt.Printf("Error parsing offset: %v\n", err)
				}
			}
		} else {
			startPos = textLength
			fmt.Printf("Using text length as start: %d\n", startPos)
		}
	}

	// Parse length (like our implementation)
	length := len(sourceText) - startPos + 1
	fmt.Printf("Calculated length: %d - %d + 1 = %d\n", len(sourceText), startPos, length)

	// Extract substring (like our implementation)
	actualSubstring := ""

	if len(sourceText) > 0 {
		effectiveStart := startPos
		effectiveLength := length

		fmt.Printf("Initial: effectiveStart=%d, effectiveLength=%d\n", effectiveStart, effectiveLength)

		// Handle startPos < 1
		if startPos < 1 {
			adjustment := 1 - startPos
			effectiveLength = length - adjustment
			effectiveStart = 1
			fmt.Printf("Adjusted for startPos < 1: effectiveStart=%d, effectiveLength=%d\n", effectiveStart, effectiveLength)
		}

		// Extract if valid
		if effectiveLength > 0 && effectiveStart <= len(sourceText) {
			start := effectiveStart - 1 // Convert to 0-based
			end := start + effectiveLength

			fmt.Printf("Go indices: start=%d, end=%d\n", start, end)

			if end > len(sourceText) {
				end = len(sourceText)
				fmt.Printf("Adjusted end to: %d\n", end)
			}

			if start >= 0 && start < len(sourceText) && end > start {
				actualSubstring = sourceText[start:end]
				fmt.Printf("Extracted substring: '%s'\n", actualSubstring)
			} else {
				fmt.Printf("Invalid indices: start=%d, end=%d, textLen=%d\n", start, end, len(sourceText))
			}
		} else {
			fmt.Printf("Skipped extraction: effectiveLength=%d, effectiveStart=%d, textLen=%d\n",
				effectiveLength, effectiveStart, len(sourceText))
		}
	}

	return actualSubstring, fmt.Sprintf("startPos=%d, length=%d", startPos, length)
}

func main() {
	fmt.Println("=== ROOT CAUSE: SUBSTRING EXTRACTION LOGIC ===")

	testCases := []struct {
		text     string
		expected string
	}{
		{"ShortText", "Text"},        // From position 6: 'Text'
		{"VeryLongTextHere", "Here"}, // From position 13: 'Here'
		{"Mid", ""},                  // From position 0: invalid
	}

	for _, tc := range testCases {
		fmt.Printf("\n=== Testing: '%s' ===\n", tc.text)

		result, debug := debugSubstringExtraction(tc.text, "string-length(text()) - 3")

		fmt.Printf("Expected: '%s'\n", tc.expected)
		fmt.Printf("Got: '%s'\n", result)
		fmt.Printf("Debug: %s\n", debug)

		if result == tc.expected {
			fmt.Printf("✅ CORRECT\n")
		} else {
			fmt.Printf("❌ WRONG\n")
		}
	}
}
