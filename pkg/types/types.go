package types

// NodeType represents the type of HTML/XML node
type NodeType int

const (
	ElementNode               NodeType = 1
	AttributeNode             NodeType = 2
	TextNode                  NodeType = 3
	CDATASectionNode          NodeType = 4
	EntityReferenceNode       NodeType = 5
	EntityNode                NodeType = 6
	ProcessingInstructionNode NodeType = 7
	CommentNode               NodeType = 8
	DocumentNode              NodeType = 9
	DocumentTypeNode          NodeType = 10
	DocumentFragmentNode      NodeType = 11
	NotationNode              NodeType = 12
)

// Node represents an HTML/XML node with location tracking
type Node struct {
	Type           NodeType          `json:"type"`
	Name           string            `json:"name"`
	Value          string            `json:"value,omitempty"`
	TextContent    string            `json:"text_content,omitempty"`
	Attributes     map[string]string `json:"attributes,omitempty"`
	AttributeOrder []string          `json:"-"` // Track document order of attributes
	Children       []*Node           `json:"children,omitempty"`
	Parent         *Node             `json:"-"` // Avoid circular JSON
	StartPos       int               `json:"start_pos"`
	EndPos         int               `json:"end_pos"`
	ContentStart   int               `json:"content_start,omitempty"` // Start of inner content (after opening tag)
	ContentEnd     int               `json:"content_end,omitempty"`   // End of inner content (before closing tag)
	StartLine      int               `json:"start_line"`
	StartColumn    int               `json:"start_column"`
	EndLine        int               `json:"end_line"`
	EndColumn      int               `json:"end_column"`
	SourceLength   int               `json:"source_length"`
}

// LocationInfo holds detailed position information
type LocationInfo struct {
	StartOffset int `json:"start_offset"`
	EndOffset   int `json:"end_offset"`
	StartLine   int `json:"start_line"`
	StartCol    int `json:"start_col"`
	EndLine     int `json:"end_line"`
	EndCol      int `json:"end_col"`
}

// XPathAxis represents XPath axes
type XPathAxis string

const (
	AxisChild            XPathAxis = "child"
	AxisDescendant       XPathAxis = "descendant"
	AxisParent           XPathAxis = "parent"
	AxisAncestor         XPathAxis = "ancestor"
	AxisFollowingSibling XPathAxis = "following-sibling"
	AxisPrecedingSibling XPathAxis = "preceding-sibling"
	AxisFollowing        XPathAxis = "following"
	AxisPreceding        XPathAxis = "preceding"
	AxisAttribute        XPathAxis = "attribute"
	AxisNamespace        XPathAxis = "namespace"
	AxisSelf             XPathAxis = "self"
	AxisDescendantOrSelf XPathAxis = "descendant-or-self"
	AxisAncestorOrSelf   XPathAxis = "ancestor-or-self"
)

// XPathFunction represents XPath functions
type XPathFunction string

const (
	FuncText           XPathFunction = "text"
	FuncNode           XPathFunction = "node"
	FuncPosition       XPathFunction = "position"
	FuncLast           XPathFunction = "last"
	FuncCount          XPathFunction = "count"
	FuncName           XPathFunction = "name"
	FuncLocalName      XPathFunction = "local-name"
	FuncNamespaceURI   XPathFunction = "namespace-uri"
	FuncString         XPathFunction = "string"
	FuncNumber         XPathFunction = "number"
	FuncBoolean        XPathFunction = "boolean"
	FuncNot            XPathFunction = "not"
	FuncStartsWith     XPathFunction = "starts-with"
	FuncContains       XPathFunction = "contains"
	FuncSubstring      XPathFunction = "substring"
	FuncNormalizeSpace XPathFunction = "normalize-space"
)

// XPathOperator represents XPath operators
type XPathOperator string

const (
	OpEqual              XPathOperator = "="
	OpNotEqual           XPathOperator = "!="
	OpLessThan           XPathOperator = "<"
	OpGreaterThan        XPathOperator = ">"
	OpLessThanOrEqual    XPathOperator = "<="
	OpGreaterThanOrEqual XPathOperator = ">="
	OpAnd                XPathOperator = "and"
	OpOr                 XPathOperator = "or"
	OpNot                XPathOperator = "not"
	OpPlus               XPathOperator = "+"
	OpMinus              XPathOperator = "-"
	OpMultiply           XPathOperator = "*"
	OpDiv                XPathOperator = "div"
	OpMod                XPathOperator = "mod"
)

// ParsedXPath represents a parsed XPath expression
type ParsedXPath struct {
	Steps      []XPathStep    `json:"steps"`
	IsAbsolute bool           `json:"is_absolute"`
	Union      []*ParsedXPath `json:"union,omitempty"` // For union expressions like //h1 | //h2
}

// XPathStep represents a single step in an XPath expression
type XPathStep struct {
	Axis       XPathAxis        `json:"axis"`
	NodeTest   string           `json:"node_test"`
	Predicates []XPathPredicate `json:"predicates,omitempty"`
}

// XPathPredicate represents an XPath predicate
type XPathPredicate struct {
	Expression string      `json:"expression"`
	Parsed     interface{} `json:"parsed,omitempty"`
}
