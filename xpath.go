package xpath

import (
	"fmt"
	"strings"

	"github.com/reclaimprotocol/xpath-go/internal/evaluator"
	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Result represents an XPath query result with location tracking
type Result struct {
	Value         string            `json:"value"`
	NodeName      string            `json:"nodeName"`
	NodeType      int               `json:"nodeType"`
	Attributes    map[string]string `json:"attributes,omitempty"`
	StartLocation int               `json:"startLocation"`
	EndLocation   int               `json:"endLocation"`
	Path          string            `json:"path"`
	TextContent   string            `json:"textContent"`
}

// XPath represents a compiled XPath expression
type XPath struct {
	expression string
	evaluator  *evaluator.Evaluator
}

// Options for XPath evaluation
type Options struct {
	IncludeLocation bool   `json:"include_location"`
	OutputFormat    string `json:"output_format"` // "nodes", "values", "paths"
}

// Query evaluates an XPath expression against HTML/XML content
func Query(xpathExpr, content string) ([]Result, error) {
	return QueryWithOptions(xpathExpr, content, Options{
		IncludeLocation: true,
		OutputFormat:    "nodes",
	})
}

// QueryWithOptions evaluates XPath with custom options
func QueryWithOptions(xpathExpr, content string, opts Options) ([]Result, error) {
	// Input validation
	if strings.TrimSpace(xpathExpr) == "" {
		return nil, fmt.Errorf("xpath expression cannot be empty")
	}
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	// Create evaluator and evaluate XPath
	eval := evaluator.NewEvaluator()
	nodes, err := eval.Evaluate(xpathExpr, content)
	if err != nil {
		return nil, err
	}

	// Convert nodes to results
	return convertNodesToResults(nodes, opts), nil
}

// Compile pre-compiles an XPath expression for repeated use
func Compile(xpathExpr string) (*XPath, error) {
	if strings.TrimSpace(xpathExpr) == "" {
		return nil, fmt.Errorf("xpath expression cannot be empty")
	}

	return &XPath{
		expression: xpathExpr,
		evaluator:  evaluator.NewEvaluator(),
	}, nil
}

// Evaluate uses a pre-compiled XPath expression
func (x *XPath) Evaluate(content string) ([]Result, error) {
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("content cannot be empty")
	}

	nodes, err := x.evaluator.Evaluate(x.expression, content)
	if err != nil {
		return nil, err
	}

	return convertNodesToResults(nodes, Options{
		IncludeLocation: true,
		OutputFormat:    "nodes",
	}), nil
}

// GetExpression returns the original XPath expression
func (x *XPath) GetExpression() string {
	return x.expression
}

// convertNodesToResults converts internal nodes to public result format
func convertNodesToResults(nodes []types.Node, opts Options) []Result {
	var results []Result

	for _, node := range nodes {
		result := Result{
			Value:         node.Value,
			NodeName:      node.Name,
			NodeType:      int(node.Type),
			Attributes:    node.Attributes,
			StartLocation: node.StartPos,
			EndLocation:   node.EndPos,
			Path:          generateNodePath(&node),
			TextContent:   node.TextContent,
		}

		// Handle different output formats
		switch opts.OutputFormat {
		case "values":
			if result.TextContent != "" {
				result.Value = result.TextContent
			}
		case "paths":
			result.Value = result.Path
		default: // "nodes"
			if result.Value == "" && result.TextContent != "" {
				result.Value = result.TextContent
			}
		}

		results = append(results, result)
	}

	return results
}

// generateNodePath generates an XPath-like path for a node
func generateNodePath(node *types.Node) string {
	if node.Parent == nil {
		return "/" + node.Name
	}

	parentPath := generateNodePath(node.Parent)
	if parentPath == "/" {
		parentPath = ""
	}

	// Add position if there are siblings with the same name
	position := 1
	if node.Parent != nil {
		for _, sibling := range node.Parent.Children {
			if sibling.Name == node.Name {
				if sibling == node {
					break
				}
				position++
			}
		}
	}

	if position > 1 || hasSiblingsWithSameName(node) {
		return fmt.Sprintf("%s/%s[%d]", parentPath, node.Name, position)
	}

	return fmt.Sprintf("%s/%s", parentPath, node.Name)
}

// hasSiblingsWithSameName checks if a node has siblings with the same name
func hasSiblingsWithSameName(node *types.Node) bool {
	if node.Parent == nil {
		return false
	}

	count := 0
	for _, sibling := range node.Parent.Children {
		if sibling.Name == node.Name {
			count++
		}
	}

	return count > 1
}

// EnableTrace enables verbose trace logging for debugging XPath evaluation
func EnableTrace() {
	evaluator.EnableTrace()
}

// DisableTrace disables trace logging
func DisableTrace() {
	evaluator.DisableTrace()
}
