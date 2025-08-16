package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test exactly what's happening in matchesNodeTest with predicates
	html := `<html><body>
		<div class="content">
			<p>Target paragraph</p>
		</div>
	</body></html>`
	
	fmt.Println("=== Testing matchesNodeTest with predicates ===")
	fmt.Println("HTML Structure: body > div.content > p")
	fmt.Println()
	
	// Get the nodes first to see the structure
	fmt.Println("1. Getting all divs:")
	divs, _ := xpath.Query("//div", html)
	for i, div := range divs {
		fmt.Printf("   Div %d: name='%s', class='%s'\n", i+1, div.NodeName, div.Attributes["class"])
	}
	fmt.Println()
	
	fmt.Println("2. Testing direct div selection with predicate:")
	result1, _ := xpath.Query("//div[@class='content']", html)
	fmt.Printf("   Query: //div[@class='content'] -> Results: %d\n", len(result1))
	fmt.Println()
	
	fmt.Println("3. Getting all p elements:")
	ps, _ := xpath.Query("//p", html)
	for i, p := range ps {
		fmt.Printf("   P %d: text='%s'\n", i+1, p.TextContent)
	}
	fmt.Println()
	
	fmt.Println("4. Testing p with simple ancestor:")
	result2, _ := xpath.Query("//p[ancestor::div]", html)
	fmt.Printf("   Query: //p[ancestor::div] -> Results: %d\n", len(result2))
	fmt.Println()
	
	fmt.Println("5. Testing p with ancestor predicate (SHOULD work but doesn't):")
	result3, _ := xpath.Query("//p[ancestor::div[@class='content']]", html)
	fmt.Printf("   Query: //p[ancestor::div[@class='content']] -> Results: %d\n", len(result3))
	fmt.Println()
	
	// Let's also try this broken down
	fmt.Println("6. Step by step manual check:")
	fmt.Println("   - We know the p exists")
	fmt.Println("   - We know the div.content exists") 
	fmt.Println("   - We know p[ancestor::div] works")
	fmt.Println("   - Therefore the issue MUST be in the predicate matching")
	fmt.Println("   - Specifically: matchesNodeTest(div_node, 'div[@class=\"content\"]')")
}