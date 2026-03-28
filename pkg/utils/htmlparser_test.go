package utils

import (
	"testing"
	"github.com/reclaimprotocol/xpath-go/pkg/types"
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
