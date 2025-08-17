package main

import (
	"fmt"
	"strings"
	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	fmt.Println("🔍 DEBUGGING ACTUAL FAILING CASES WITH TRACE MODE")
	fmt.Println(strings.Repeat("=", 60))

	// The 7 failing cases from the comprehensive comparison
	testCases := []struct {
		name  string
		html  string
		xpath string
		jsResult int
		desc  string
	}{
		{
			name:     "Complex boolean logic",
			html:     `<div id="a" class="x y">A</div><div id="b" class="y">B</div><div id="c" class="x z">C</div>`,
			xpath:    `//div[(@id='a' or @id='c') and contains(@class, 'x')]`,
			jsResult: 2,
			desc:     "Should find divs with id 'a' OR 'c' AND containing class 'x'",
		},
		{
			name:     "Position in filtered set",
			html:     `<div><span class="item">First</span><span>Middle</span><span class="item">Second</span><span class="item">Third</span></div>`,
			xpath:    `//span[@class='item'][position() mod 2 = 0]`,
			jsResult: 2,
			desc:     "Should find even-positioned spans in filtered set",
		},
		{
			name:     "Conditional attribute selection",
			html:     `<a href="http://example.com">Link1</a><a href="/local" title="Local">Link2</a><a href="https://test.com" title="Test">Link3</a>`,
			xpath:    `//a[@href and (@title or starts-with(@href, 'http'))]`,
			jsResult: 3,
			desc:     "Should find links with href AND (title OR href starting with 'http')",
		},
		{
			name:     "Class list manipulation",
			html:     `<div class="primary active">A</div><div class="primary inactive">B</div><div class="secondary active">C</div>`,
			xpath:    `//div[contains(@class, 'primary') and contains(@class, 'active') and not(contains(@class, 'inactive'))]`,
			jsResult: 1,
			desc:     "Should find div with 'primary' AND 'active' but NOT 'inactive'",
		},
		{
			name:     "Document structure validation",
			html:     `<html><head><title>Test</title><meta charset="utf-8"/></head><body><main>Content</main></body></html>`,
			xpath:    `/html[head/title and head/meta[@charset] and body/main]`,
			jsResult: 1,
			desc:     "Should validate complete document structure",
		},
		{
			name:     "Substring length comparison",
			html:     `<span>First text</span><span>Second item</span><span>Third content</span>`,
			xpath:    `//span[substring(text(), 1, 3) = 'Fir' or substring(text(), 1, 3) = 'Thi']`,
			jsResult: 2,
			desc:     "Should find spans starting with 'Fir' OR 'Thi'",
		},
		{
			name:     "Substring edge cases",
			html:     `<div>Content here</div><div>Another text</div><div>Some data</div>`,
			xpath:    `//div[string-length(text()) > 0 and substring(text(), 1, 1) = 'C']`,
			jsResult: 1,
			desc:     "Should find div with non-empty text starting with 'C'",
		},
	}

	// Enable trace mode for detailed debugging
	xpath.EnableTrace()

	for i, tc := range testCases {
		fmt.Printf("\n%d. %s\n", i+1, tc.name)
		fmt.Printf("   %s\n", tc.desc)
		fmt.Printf("   XPath: %s\n", tc.xpath)
		fmt.Printf("   Expected: %d matches\n\n", tc.jsResult)
		
		fmt.Println("--- TRACE OUTPUT ---")
		results, err := xpath.Query(tc.xpath, tc.html)
		
		fmt.Printf("\n--- RESULTS ---\n")
		if err != nil {
			fmt.Printf("❌ ERROR: %v\n", err)
		} else {
			fmt.Printf("Found %d matches (expected %d):", len(results), tc.jsResult)
			if len(results) == tc.jsResult {
				fmt.Print(" ✅ PASS")
			} else {
				fmt.Print(" ❌ FAIL")
			}
			fmt.Println()
			
			for j, result := range results {
				fmt.Printf("  %d. <%s", j+1, result.NodeName)
				if len(result.Attributes) > 0 {
					for attr, val := range result.Attributes {
						fmt.Printf(" %s='%s'", attr, val)
					}
				}
				fmt.Printf(">")
				if result.TextContent != "" {
					fmt.Printf(": '%s'", result.TextContent)
				}
				fmt.Println()
			}
		}
		
		fmt.Println("\n" + strings.Repeat("-", 60))
	}

	xpath.DisableTrace()
	fmt.Println("\n🔍 Trace analysis complete.")
}