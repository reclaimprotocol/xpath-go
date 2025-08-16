package main

import (
	"fmt"
	"strings"
)

func main() {
	expr := "@class='red' or @class='blue'"
	fmt.Printf("Expression: %s\n", expr)
	fmt.Printf("Contains ' or ': %v\n", strings.Contains(expr, " or "))

	parts := strings.Split(expr, " or ")
	fmt.Printf("Split parts: %v\n", parts)
	fmt.Printf("First part: '%s'\n", strings.TrimSpace(parts[0]))
	fmt.Printf("Second part: '%s'\n", strings.TrimSpace(parts[1]))
}
