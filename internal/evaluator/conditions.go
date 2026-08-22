package evaluator

import (
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// matchesNodeTest checks if a node matches a node test
func (e *Evaluator) matchesNodeTest(node *types.Node, nodeTest string) bool {
	if nodeTest == "*" {
		return node.Type == types.ElementNode
	}
	if nodeTest == "node()" || nodeTest == "node" {
		return true
	}
	if nodeTest == "text()" {
		return node.Type == types.TextNode
	}
	if nodeTest == "comment()" {
		return node.Type == types.CommentNode
	}
	if nodeTest == "processing-instruction()" || strings.HasPrefix(nodeTest, "processing-instruction(") {
		target := ""
		if nodeTest != "processing-instruction()" {
			target = strings.TrimSuffix(strings.TrimPrefix(nodeTest, "processing-instruction("), ")")
			target = strings.Trim(target, `"'`)
		}
		return node.Type == types.ProcessingInstructionNode && (target == "" || node.Name == target)
	}

	return node.Type == types.ElementNode && node.Name == nodeTest
}
