package xpath_test

import "strings"

// documentBodyExpression migrates legacy whole-document tests that described
// source elements as Document children. HTML document parsing now always
// creates the browser-compatible html/head/body skeleton, so an absolute path
// to an ordinary source element starts at /html/body.
func documentBodyExpression(expression string) string {
	if !strings.HasPrefix(expression, `/`) || strings.HasPrefix(expression, `//`) ||
		strings.HasPrefix(expression, `/html`) || strings.HasPrefix(expression, `/node(`) ||
		strings.HasPrefix(expression, `/comment(`) || strings.HasPrefix(expression, `/processing-instruction(`) ||
		strings.HasPrefix(expression, `/text(`) {
		return expression
	}
	return `/html/body` + expression
}
