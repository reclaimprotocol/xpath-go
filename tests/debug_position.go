package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/PuerkitoBio/goquery"
	xpath "github.com/antchfx/xpath"
)

func main() {
	// Test 54: Complex table navigation
	html1 := `<html><body><table><thead><tr><th>Name</th><th>Age</th></tr></thead><tbody><tr><td>John</td><td>25</td></tr><tr><td>Jane</td><td>30</td></tr></tbody></table></body></html>`
	xpath1 := `//tbody/tr[position()>1]/td[position()=1]`

	// Test 62: Position in filtered set
	html2 := `<html><body><div><span class='item'>A</span><p>X</p><span class='item'>B</span><div>Y</div><span class='item'>C</span><span class='item'>D</span></div></body></html>`
	xpath2 := `//span[@class='item'][position() mod 2 = 0]`

	fmt.Println("=== DEBUGGING POSITION() FUNCTION ===\n")

	fmt.Println("Test 1: Table navigation")
	fmt.Printf("XPath: %s\n", xpath1)
	fmt.Printf("Expected: 1 result (Jane)\n")
	testXPath(html1, xpath1)
	fmt.Println()

	fmt.Println("Test 2: Position in filtered set")
	fmt.Printf("XPath: %s\n", xpath2)
	fmt.Printf("Expected: 2 results (B and D - even positions)\n")
	testXPath(html2, xpath2)
	fmt.Println()

	// Debug simpler position queries
	fmt.Println("=== SIMPLER POSITION TESTS ===\n")

	simpleHTML := `<html><body><ul><li>Item 1</li><li>Item 2</li><li>Item 3</li></ul></body></html>`

	// Test basic position
	fmt.Println("Basic position=2:")
	testXPath(simpleHTML, "//li[position()=2]")
	fmt.Println()

	// Test position>1
	fmt.Println("Basic position>1:")
	testXPath(simpleHTML, "//li[position()>1]")
	fmt.Println()
}

func testXPath(html, xpathQuery string) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		log.Printf("Error parsing HTML: %v", err)
		return
	}

	// Parse XPath
	expr, err := xpath.Compile(xpathQuery)
	if err != nil {
		log.Printf("Error compiling XPath: %v", err)
		return
	}

	// Create navigator from goquery document
	nav := NewGoQueryNavigator(doc.Selection)

	// Execute XPath
	iter := expr.Select(nav)

	results := []map[string]interface{}{}
	count := 0

	for iter.MoveNext() {
		count++
		node := iter.Current()

		result := map[string]interface{}{
			"nodeName":    strings.ToLower(node.LocalName()),
			"textContent": strings.TrimSpace(node.Value()),
			"nodeType":    1, // Element node
			"attributes":  map[string]string{},
		}

		results = append(results, result)

		fmt.Printf("  Result %d: <%s>%s</%s>\n", count,
			node.LocalName(), node.Value(), node.LocalName())
	}

	fmt.Printf("Found %d results\n", count)

	// Output JSON for consistency
	jsonResult := map[string]interface{}{
		"results": results,
		"count":   count,
	}
	jsonBytes, _ := json.Marshal(jsonResult)
	fmt.Printf("JSON: %s\n", jsonBytes)
}

// Navigation implementation - simplified version
type GoQueryNavigator struct {
	*goquery.Selection
	index int
}

func NewGoQueryNavigator(sel *goquery.Selection) *GoQueryNavigator {
	return &GoQueryNavigator{Selection: sel, index: 0}
}

func (g *GoQueryNavigator) NodeType() xpath.NodeType {
	return xpath.ElementNode
}

func (g *GoQueryNavigator) LocalName() string {
	if g.Length() == 0 {
		return ""
	}
	node := g.Get(0)
	return node.Data
}

func (g *GoQueryNavigator) Value() string {
	return strings.TrimSpace(g.Text())
}

func (g *GoQueryNavigator) Copy() xpath.NodeNavigator {
	return &GoQueryNavigator{Selection: g.Selection, index: g.index}
}

func (g *GoQueryNavigator) MoveToRoot() {
	g.Selection = g.Selection.First().ParentsUntil("html").Last().Parent()
}

func (g *GoQueryNavigator) MoveToParent() bool {
	parent := g.Selection.Parent()
	if parent.Length() > 0 {
		g.Selection = parent
		return true
	}
	return false
}

func (g *GoQueryNavigator) MoveToNextAttribute() bool {
	return false
}

func (g *GoQueryNavigator) MoveToChild() bool {
	children := g.Selection.Children()
	if children.Length() > 0 {
		g.Selection = children.First()
		return true
	}
	return false
}

func (g *GoQueryNavigator) MoveToFirst() bool {
	if g.Length() > 0 {
		g.Selection = g.First()
		return true
	}
	return false
}

func (g *GoQueryNavigator) MoveToNext() bool {
	next := g.Selection.Next()
	if next.Length() > 0 {
		g.Selection = next
		return true
	}
	return false
}

func (g *GoQueryNavigator) MoveToPrevious() bool {
	prev := g.Selection.Prev()
	if prev.Length() > 0 {
		g.Selection = prev
		return true
	}
	return false
}

func (g *GoQueryNavigator) MoveTo(other xpath.NodeNavigator) bool {
	return false
}
