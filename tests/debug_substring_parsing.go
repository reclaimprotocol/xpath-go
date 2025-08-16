package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Copy exact logic from our implementation
func parseSubstringArgs(argsStr string) []string {
	var args []string
	current := ""
	inQuotes := false
	quoteChar := byte(0)
	parenDepth := 0

	for i := 0; i < len(argsStr); i++ {
		c := argsStr[i]

		if !inQuotes && (c == '\'' || c == '"') {
			inQuotes = true
			quoteChar = c
			current += string(c)
		} else if inQuotes && c == quoteChar {
			inQuotes = false
			quoteChar = 0
			current += string(c)
		} else if !inQuotes && c == '(' {
			parenDepth++
			current += string(c)
		} else if !inQuotes && c == ')' {
			parenDepth--
			current += string(c)
		} else if !inQuotes && c == ',' && parenDepth == 0 {
			args = append(args, strings.TrimSpace(current))
			current = ""
		} else {
			current += string(c)
		}
	}

	if current != "" {
		args = append(args, strings.TrimSpace(current))
	}

	return args
}

func debugSubstringForText(text string) {
	fmt.Printf("\n=== DEBUGGING: '%s' ===\n", text)

	// Test argument parsing
	argsStr := "text(), string-length(text()) - 3"
	args := parseSubstringArgs(argsStr)

	fmt.Printf("Args: %v (count: %d)\n", args, len(args))

	// Calculate start position (like our implementation)
	startPos := 1
	if strings.Contains(args[1], "string-length(text())") && strings.Contains(args[1], " - ") {
		parts := strings.Split(args[1], " - ")
		fmt.Printf("Parts: %v\n", parts)
		if len(parts) == 2 {
			if offset, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
				startPos = len(text) - offset
				fmt.Printf("Calculated: len(%d) - offset(%d) = startPos(%d)\n", len(text), offset, startPos)
			}
		}
	}

	// Call our substring logic
	fmt.Printf("Args count: %d (>2? %t)\n", len(args), len(args) > 2)

	var actualSubstring string
	if len(args) > 2 {
		fmt.Println("Taking 3-argument path")
		if length, err := strconv.Atoi(strings.TrimSpace(args[2])); err == nil {
			actualSubstring = xpathSubstring(text, startPos, length)
		} else {
			actualSubstring = xpathSubstring(text, startPos, -1)
		}
	} else {
		fmt.Println("Taking 2-argument path")
		actualSubstring = xpathSubstring(text, startPos, -1)
	}

	fmt.Printf("Result: '%s'\n", actualSubstring)
	fmt.Printf("Matches 'Text'? %t\n", actualSubstring == "Text")
}

func xpathSubstring(text string, startPos int, length int) string {
	if text == "" {
		return ""
	}

	if startPos <= 0 {
		if startPos == 0 && len(text) > 0 {
			return string(text[len(text)-1])
		}
		return ""
	}

	start := startPos - 1

	if start >= len(text) {
		return ""
	}

	if length == -1 {
		return text[start:]
	}

	if length <= 0 {
		return ""
	}

	end := start + length
	if end > len(text) {
		end = len(text)
	}

	return text[start:end]
}

func main() {
	fmt.Println("=== SUBSTRING PARSING DEBUG ===")

	testTexts := []string{"ShortText", "VeryLongTextHere", "Mid"}

	for _, text := range testTexts {
		debugSubstringForText(text)
	}
}
