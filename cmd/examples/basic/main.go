package main

import (
	"fmt"
	"log"
	"os"

	"github.com/reclaimprotocol/xpath-go"
)

func main() {
	// Example HTML content
	html := `<html>
<body>
    <div id="header" class="main-header">
        <h1>Welcome to XPath Go</h1>
        <nav>
            <ul>
                <li><a href="/">Home</a></li>
                <li><a href="/about">About</a></li>
                <li><a href="/contact">Contact</a></li>
            </ul>
        </nav>
    </div>
    <div id="content" class="main-content">
        <article>
            <h2>XPath Examples</h2>
            <p>This library provides <strong>high compatibility</strong> with precise location tracking.</p>
            <p>Here are some example XPath expressions:</p>
            <ul>
                <li><code>//div[@id='header']</code> - Select header div</li>
                <li><code>//a[contains(@href, 'about')]</code> - Select about link</li>
                <li><code>//p[position()=2]</code> - Select second paragraph</li>
            </ul>
        </article>
    </div>
    <footer id="footer">
        <p>&copy; 2024 Reclaim Protocol</p>
    </footer>
</body>
</html>`

	examples := []struct {
		name  string
		xpath string
		desc  string
	}{
		{"Header Selection", "//div[@id='header']", "Select the header div element"},
		{"Navigation Links", "//nav//a", "Select all navigation links"},
		{"Paragraphs with Text", "//p[text()]", "Select paragraphs containing text"},
		{"Strong Elements", "//strong", "Select all strong elements"},
		{"Second List Item", "//ul/li[position()=2]", "Select second list item"},
		{"Elements with Class", "//*[@class]", "Select all elements with class attribute"},
	}

	fmt.Println("🎯 XPath Go - Basic Examples")
	fmt.Println("============================")
	fmt.Printf("Library Version: %s\n", xpath.Version)
	fmt.Println()

	for i, example := range examples {
		fmt.Printf("[%d] %s\n", i+1, example.name)
		fmt.Printf("    XPath: %s\n", example.xpath)
		fmt.Printf("    Description: %s\n", example.desc)

		results, err := xpath.Query(example.xpath, html)
		if err != nil {
			fmt.Printf("    ❌ Error: %v\n", err)
			continue
		}

		fmt.Printf("    ✅ Found %d result(s):\n", len(results))
		for j, result := range results {
			fmt.Printf("       [%d] Node: %s, Text: %q\n",
				j+1, result.NodeName, truncateText(result.TextContent, 50))
			if result.StartLocation > 0 || result.EndLocation > 0 {
				fmt.Printf("           Location: %d-%d\n", result.StartLocation, result.EndLocation)
			}
		}
		fmt.Println()
	}

	// Show build info if available
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		buildInfo := xpath.GetBuildInfo()
		fmt.Println("Build Information:")
		fmt.Printf("  Version: %s\n", buildInfo.Version)
		fmt.Printf("  API Version: %s\n", buildInfo.APIVersion)
		fmt.Printf("  Go Version: %s\n", buildInfo.GoVersion)
		fmt.Printf("  Platform: %s\n", buildInfo.Platform)
		fmt.Printf("  Compiler: %s\n", buildInfo.Compiler)
	}
}

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

func init() {
	// Check Go version compatibility
	if err := xpath.CheckGoVersion(); err != nil {
		log.Fatalf("Go version compatibility check failed: %v", err)
	}
}
