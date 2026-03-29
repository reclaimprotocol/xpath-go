package utils

import (
	"github.com/reclaimprotocol/xpath-go/pkg/types"
	"testing"
)

func TestParseForgivingHTML(t *testing.T) {
	htmlContent := `<html>
<head>
    <link rel="shortcut icon" href="/MemberPassBook/static/favicon.ico" <="" head="">
</head>
<body>
    <div class="collapse sidebar navbar-collapse main-nav" id="main_nav">
        <ul class="navbar-nav navbar-nav-nav">
        </ul>
        <ul class="navbar-nav ms-auto">
            <span class="navbar-text navbar-text-nav custom-user-label">
                GADDAMEDI VAMSHI KRISHNA
            </span>
        </ul>
    </div>
</body>
</html>`

	parser := NewHTMLParser()
	node, err := parser.Parse(htmlContent)
	if err != nil {
		t.Fatalf("Expected parsing to succeed, but got error: %v", err)
	}

	if node == nil {
		t.Fatalf("Expected non-nil document node")
	}

	// Make sure html -> body -> div -> span is parsed
	htmlNode := findChild(node, "html")
	if htmlNode == nil {
		t.Fatalf("Could not find html node")
	}

	bodyNode := findChild(htmlNode, "body")
	if bodyNode == nil {
		t.Fatalf("Could not find body node")
	}

	divNode := findChild(bodyNode, "div")
	if divNode == nil {
		t.Fatalf("Could not find div node")
	}

	ulNodes := findChildren(divNode, "ul")
	if len(ulNodes) != 2 {
		t.Fatalf("Expected 2 ul nodes, got %d", len(ulNodes))
	}

	spanNode := findChild(ulNodes[1], "span")
	if spanNode == nil {
		t.Fatalf("Could not find span node inside second ul")
	}
}

func TestValidContentLoss(t *testing.T) {
	// Test case 1: Valid siblings should not be lost due to one malformed element
	htmlContent := `<ul><li>a</li><li <=""></li><li>c</li></ul>`

	parser := NewHTMLParser()
	node, err := parser.Parse(htmlContent)
	if err != nil {
		t.Fatalf("Expected parsing to succeed, but got error: %v", err)
	}

	ulNode := findChild(node, "ul")
	if ulNode == nil {
		t.Fatalf("Could not find ul node")
	}

	liNodes := findChildren(ulNode, "li")
	// We expect to preserve valid content: first <li>a</li> and third <li>c</li>
	// Currently fails - the entire ul is discarded
	if len(liNodes) < 2 {
		t.Errorf("Expected at least 2 valid li nodes, got %d. Valid content was lost!", len(liNodes))
		t.Logf("UL node has %d children total", len(ulNode.Children))
	}
}

func TestWrongResyncPoint(t *testing.T) {
	// Test case 2: Parser should resync at the correct closing tag, not nested ones
	htmlContent := `<div><span <=""><div></div><p>keep</p></div>`

	parser := NewHTMLParser()
	node, err := parser.Parse(htmlContent)
	if err != nil {
		t.Fatalf("Expected parsing to succeed, but got error: %v", err)
	}

	divNode := findChild(node, "div")
	if divNode == nil {
		t.Fatalf("Could not find outer div node")
	}

	// Check if we have the <p>keep</p> element
	pNode := findChild(divNode, "p")
	if pNode == nil {
		t.Errorf("Expected to find <p> element with 'keep' text after recovery")
		t.Logf("Outer div has %d children", len(divNode.Children))
		for i, child := range divNode.Children {
			t.Logf("Child %d: %s (type: %d)", i, child.Name, child.Type)
		}
	} else {
		// Verify the text content
		if pNode.TextContent != "keep" {
			t.Errorf("Expected <p> text to be 'keep', got '%s'", pNode.TextContent)
		}
	}
}

func findChild(parent *types.Node, name string) *types.Node {
	for _, child := range parent.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

func findChildren(parent *types.Node, name string) []*types.Node {
	var result []*types.Node
	for _, child := range parent.Children {
		if child.Name == name {
			result = append(result, child)
		}
	}
	return result
}
