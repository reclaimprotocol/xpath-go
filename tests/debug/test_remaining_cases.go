package main

import (
	"fmt"
	"strings"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Test cases from the failing test indices
	testCases := []struct {
		name  string
		html  string
		xpath string
		desc  string
	}{
		{
			name:  "Empty vs non-empty elements (FIXED)",
			html:  `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`,
			xpath: `//div[normalize-space(text())='' and not(*)]`,
			desc:  "Should find empty and space-only divs without child elements",
		},
		{
			name:  "Position in filtered set",
			html:  `<ul><li class="item">A</li><li>B</li><li class="item">C</li><li>D</li><li class="item">E</li></ul>`,
			xpath: `//li[@class='item'][position() mod 2 = 1]`,
			desc:  "Should find odd-positioned items in the filtered set",
		},
		{
			name:  "Class list manipulation",
			html:  `<div class="nav active"><span class="icon">Icon</span><a href="#" class="link active">Link</a></div>`,
			xpath: `//div[contains(@class, 'nav') and contains(@class, 'active')]//*[contains(@class, 'active')]`,
			desc:  "Should find elements with active class inside nav div",
		},
		{
			name:  "Document structure validation",
			html:  `<html><head><meta charset="utf-8"/><title>Test</title></head><body><h1>Header</h1></body></html>`,
			xpath: `//head[meta[@charset] and title]/following-sibling::body[h1]`,
			desc:  "Should validate document structure with meta charset",
		},
		{
			name:  "Substring length comparison",
			html:  `<div><p title="Short">Text1</p><p title="A very long title text">Text2</p><p title="Medium title">Text3</p></div>`,
			xpath: `//p[string-length(@title) > 10 or substring(@title, 1, 5) = 'Short']`,
			desc:  "Should find paragraphs with long titles OR starting with 'Short'",
		},
		{
			name:  "Substring edge cases",
			html:  `<data><item value="prefix_test_suffix">A</item><item value="test_only">B</item><item value="prefix_other">C</item></data>`,
			xpath: `//item[substring-after(@value, 'prefix_') and substring-before(@value, '_suffix')]`,
			desc:  "Should find items with specific substring patterns",
		},
	}

	fmt.Println("Testing remaining XPath compatibility cases:")
	fmt.Println(strings.Repeat("=", 50))

	for i, tc := range testCases {
		fmt.Printf("\n%d. %s\n", i+1, tc.name)
		fmt.Printf("   %s\n", tc.desc)
		fmt.Printf("   XPath: %s\n", tc.xpath)
		
		results, err := xpath.Query(tc.xpath, tc.html)
		if err != nil {
			fmt.Printf("   ❌ ERROR: %v\n", err)
			continue
		}
		
		fmt.Printf("   ✓ Found %d matches", len(results))
		if len(results) > 0 && len(results) <= 3 {
			fmt.Print(" - ")
			for j, result := range results {
				if j > 0 {
					fmt.Print(", ")
				}
				fmt.Printf("<%s>", result.NodeName)
				if result.TextContent != "" {
					fmt.Printf(":'%s'", result.TextContent)
				}
			}
		}
		fmt.Println()
	}
}