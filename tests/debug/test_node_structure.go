package main

import (
	"fmt"
	"github.com/reclaimprotocol/xpath-go/pkg/utils"
	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func main() {
	html := `<html><body><div></div><div> </div><div><span></span></div><div>Content</div></body></html>`
	
	fmt.Println("Testing Internal Node Structure")
	fmt.Println("HTML:", html)
	fmt.Println()
	
	// Parse HTML using the internal parser
	parser := utils.NewHTMLParser()
	root, err := parser.Parse(html)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	// Find all div nodes
	divs := findDivs(root)
	fmt.Printf("Found %d div nodes:\n", len(divs))
	
	for i, div := range divs {
		fmt.Printf("Div %d: Name='%s', Text='%s' (len=%d)\n", 
			i+1, div.Name, div.TextContent, len(div.TextContent))
		fmt.Printf("  Children: %d\n", len(div.Children))
		for j, child := range div.Children {
			fmt.Printf("    Child %d: Type=%d, Name='%s', Value='%s'\n", 
				j+1, child.Type, child.Name, child.Value)
		}
		fmt.Println()
	}
}

func findDivs(node *types.Node) []*types.Node {
	var divs []*types.Node
	
	if node.Name == "div" {
		divs = append(divs, node)
	}
	
	for _, child := range node.Children {
		divs = append(divs, findDivs(child)...)
	}
	
	return divs
}