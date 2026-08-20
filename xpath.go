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
	NamespaceURI  string            `json:"namespaceURI,omitempty"`
	LocalName     string            `json:"localName,omitempty"`
	Prefix        string            `json:"prefix,omitempty"`
	Attributes    map[string]string `json:"attributes,omitempty"`
	StartLocation int               `json:"startLocation"`
	EndLocation   int               `json:"endLocation"`
	ContentStart  int               `json:"contentStart,omitempty"` // Start of inner content (after opening tag)
	ContentEnd    int               `json:"contentEnd,omitempty"`   // End of inner content (before closing tag)
	Path          string            `json:"path"`
	TextContent   string            `json:"textContent"`
}

// XPath represents a compiled XPath expression
type XPath struct {
	expression string
}

// Options for XPath evaluation
type Options struct {
	IncludeLocation  bool   `json:"include_location"`
	OutputFormat     string `json:"output_format"`     // "nodes", "values", "paths"
	ContentsOnly     bool   `json:"contents_only"`     // Extract only inner content between tags
	Debug            bool   `json:"debug"`             // Enable verbose debug logging
	ScriptingEnabled bool   `json:"scripting_enabled"` // Parse noscript contents as RAWTEXT when true
	// Charset is the response-body character set used for HTML parsing and
	// XPath evaluation. Empty means tolerant UTF-8; labels use HTML/WHATWG
	// aliases (for example, ISO-8859-1 resolves to Windows-1252).
	Charset string `json:"charset"`
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
	return QueryBytesWithOptions(xpathExpr, []byte(content), opts)
}

// QueryBytes evaluates an XPath expression against raw response bytes. It is
// equivalent to Query(string(content)), but makes byte-oriented usage explicit.
func QueryBytes(xpathExpr string, content []byte) ([]Result, error) {
	return QueryBytesWithOptions(xpathExpr, content, Options{
		IncludeLocation: true,
		OutputFormat:    "nodes",
	})
}

// QueryBytesWithOptions evaluates an XPath expression against raw response
// bytes. XPath sees a decoded Unicode DOM while Result locations and full-node
// values continue to refer to the original byte stream.
func QueryBytesWithOptions(xpathExpr string, content []byte, opts Options) ([]Result, error) {
	// Input validation
	if strings.TrimSpace(xpathExpr) == "" {
		return nil, fmt.Errorf("xpath expression cannot be empty")
	}
	decodedContent, mapper, err := decodeForXPath(content, opts.Charset)
	if err != nil {
		return nil, err
	}
	// Create evaluator and evaluate XPath
	if opts.Debug {
		EnableTrace()
	}
	eval := evaluator.NewEvaluator()
	eval.SetScriptingEnabled(opts.ScriptingEnabled)
	nodes, err := eval.EvaluateWithDocument(xpathExpr, decodedContent, mapper.remapDocumentLocations)
	if err != nil {
		return nil, err
	}

	// Convert nodes to results
	return convertNodesToResults(nodes, opts, string(content)), nil
}

// Compile pre-compiles an XPath expression for repeated use
func Compile(xpathExpr string) (*XPath, error) {
	if strings.TrimSpace(xpathExpr) == "" {
		return nil, fmt.Errorf("xpath expression cannot be empty")
	}

	return &XPath{
		expression: xpathExpr,
	}, nil
}

// Evaluate uses a pre-compiled XPath expression
func (x *XPath) Evaluate(content string) ([]Result, error) {
	return x.EvaluateWithOptions(content, Options{
		IncludeLocation: true,
		OutputFormat:    "nodes",
	})
}

// EvaluateWithOptions evaluates a pre-compiled XPath expression using the
// supplied response-body decoding options.
func (x *XPath) EvaluateWithOptions(content string, opts Options) ([]Result, error) {
	return x.EvaluateBytesWithOptions([]byte(content), opts)
}

// EvaluateBytesWithOptions evaluates a pre-compiled XPath expression against
// raw response bytes. See QueryBytesWithOptions for the location contract.
func (x *XPath) EvaluateBytesWithOptions(content []byte, opts Options) ([]Result, error) {
	if x == nil {
		return nil, fmt.Errorf("XPath is nil")
	}
	decodedContent, mapper, err := decodeForXPath(content, opts.Charset)
	if err != nil {
		return nil, err
	}
	if opts.Debug {
		EnableTrace()
	}
	// Evaluator owns mutable parser, tree-builder, and axis state. Keep it
	// scoped to this call so a compiled expression remains safe to share
	// between concurrent evaluations with different options/documents.
	eval := evaluator.NewEvaluator()
	eval.SetScriptingEnabled(opts.ScriptingEnabled)
	nodes, err := eval.EvaluateWithDocument(x.expression, decodedContent, mapper.remapDocumentLocations)
	if err != nil {
		return nil, err
	}

	return convertNodesToResults(nodes, opts, string(content)), nil
}

// GetExpression returns the original XPath expression
func (x *XPath) GetExpression() string {
	return x.expression
}

// convertNodesToResults converts internal nodes to public result format
func convertNodesToResults(nodes []types.Node, opts Options, originalContent string) []Result {
	var results []Result

	for _, node := range nodes {
		result := Result{
			Value:        node.Value,
			NodeName:     node.Name,
			NodeType:     int(node.Type),
			NamespaceURI: node.NamespaceURI,
			LocalName:    node.LocalName,
			Prefix:       node.Prefix,
			Attributes:   node.Attributes,
			Path:         generateNodePath(&node),
			TextContent:  node.TextContent,
			ContentStart: node.ContentStart,
			ContentEnd:   node.ContentEnd,
		}

		// Handle contentsOnly option - adjust positions and value based on extraction mode
		if opts.ContentsOnly {
			// Only elements have an inner source range. For non-container nodes
			// such as text, comments, and processing instructions, retain the
			// node's full source range while exposing its DOM text value.
			if node.Type == types.ElementNode {
				result.StartLocation = node.ContentStart
				result.EndLocation = node.ContentEnd
			} else {
				result.StartLocation = node.StartPos
				result.EndLocation = node.EndPos
			}
			// For content-only mode, value should be just the text content
			result.Value = node.TextContent
		} else {
			// Use full element positions (including tags)
			result.StartLocation = node.StartPos
			result.EndLocation = node.EndPos
			// For full mode, value should include the HTML markup
			if node.Type == types.DocumentTypeNode {
				// Browser DocumentType.nodeValue and textContent are null/empty;
				// its source range remains available through the location fields.
				result.Value = ""
			} else if node.StartPos < len(originalContent) && node.EndPos <= len(originalContent) && node.EndPos > node.StartPos {
				result.Value = originalContent[node.StartPos:node.EndPos]
			} else if result.Value == "" && result.TextContent != "" {
				result.Value = result.TextContent
			}
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
			// Value is already set above based on extraction mode
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
