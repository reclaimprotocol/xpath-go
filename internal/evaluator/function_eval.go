package evaluator

import (
	"strconv"
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// evaluateFunction evaluates a function call using the new parser
func (e *Evaluator) evaluateFunction(fn *FunctionCall, node *types.Node) string {
	Trace("evaluateFunction: %s with %d arguments", fn.Name, len(fn.Arguments))

	// For all function evaluations, if an argument is a node-set, its string value
	// should have the NodeSetPrefix stripped when used as a string.
	// But we handle this inside each case as needed, or globally if appropriate.
	switch fn.Name {
	case "string-length":
		text := ""
		if len(fn.Arguments) == 0 {
			text = node.TextContent
		} else if len(fn.Arguments) == 1 {
			text = strings.TrimPrefix(fn.Arguments[0].Evaluate(node, e), NodeSetPrefix)
		} else {
			return "0"
		}
		result := strconv.Itoa(len(text))
		Trace("string-length('%s') = %s", text, result)
		return result

	case "normalize-space":
		text := ""
		if len(fn.Arguments) == 0 {
			text = node.TextContent
		} else if len(fn.Arguments) == 1 {
			text = strings.TrimPrefix(fn.Arguments[0].Evaluate(node, e), NodeSetPrefix)
		} else {
			return ""
		}
		// Normalize whitespace: trim and collapse multiple spaces
		normalized := strings.Join(strings.Fields(text), " ")
		Trace("normalize-space('%s') = '%s'", text, normalized)
		return normalized

	case "substring":
		if len(fn.Arguments) < 2 || len(fn.Arguments) > 3 {
			return ""
		}

		text := strings.TrimPrefix(fn.Arguments[0].Evaluate(node, e), NodeSetPrefix)
		startStr := strings.TrimPrefix(fn.Arguments[1].Evaluate(node, e), NodeSetPrefix)

		startFloat, err := strconv.ParseFloat(startStr, 64)
		if err != nil {
			return ""
		}
		start := int(startFloat)

		// XPath is 1-based, Go is 0-based
		start = start - 1
		if start < 0 {
			start = 0
		}
		if start >= len(text) {
			return ""
		}

		if len(fn.Arguments) == 2 {
			// substring(string, start) - from start to end
			result := text[start:]
			Trace("substring('%s', %d) = '%s'", text, start+1, result)
			return result
		} else {
			// substring(string, start, length)
			lengthStr := strings.TrimPrefix(fn.Arguments[2].Evaluate(node, e), NodeSetPrefix)
			lengthFloat, err := strconv.ParseFloat(lengthStr, 64)
			if err != nil || lengthFloat <= 0 {
				return ""
			}
			length := int(lengthFloat)

			end := start + length
			if end > len(text) {
				end = len(text)
			}

			result := text[start:end]
			Trace("substring('%s', %d, %d) = '%s'", text, start+1, length, result)
			return result
		}

	case "contains":
		if len(fn.Arguments) != 2 {
			return "false"
		}
		text := strings.TrimPrefix(fn.Arguments[0].Evaluate(node, e), NodeSetPrefix)
		search := strings.TrimPrefix(fn.Arguments[1].Evaluate(node, e), NodeSetPrefix)
		if strings.Contains(text, search) {
			return "true"
		}
		return "false"

	case "starts-with":
		if len(fn.Arguments) != 2 {
			return "false"
		}
		text := strings.TrimPrefix(fn.Arguments[0].Evaluate(node, e), NodeSetPrefix)
		prefix := strings.TrimPrefix(fn.Arguments[1].Evaluate(node, e), NodeSetPrefix)
		if strings.HasPrefix(text, prefix) {
			return "true"
		}
		return "false"

	case "concat":
		var result strings.Builder
		for _, arg := range fn.Arguments {
			res := arg.Evaluate(node, e)
			result.WriteString(strings.TrimPrefix(res, NodeSetPrefix))
		}
		return result.String()

	case "not":
		if len(fn.Arguments) != 1 {
			return "false"
		}
		result := fn.Arguments[0].Evaluate(node, e)
		if isTruthy(result) {
			return "false"
		}
		return "true"

	case "node":
		// node() check - traditionally matches any node
		// In predicate [node()], it means has any child node
		if len(node.Children) > 0 {
			return "true"
		}
		return "false"

	case "count":
		if len(fn.Arguments) != 1 {
			return "0"
		}
		arg := fn.Arguments[0]
		// General case for path or other expression
		res := arg.Evaluate(node, e)
		if isTruthy(res) {
			if path, ok := arg.(*PathExpression); ok {
				// Evaluate path but get result nodes
				currentNodes := []*types.Node{node}
				if path.IsAbsolute {
					root := node
					for root.Parent != nil {
						root = root.Parent
					}
					if path.IsDeep {
						currentNodes = e.getDescendantNodes(root, true)
					} else {
						currentNodes = []*types.Node{root}
					}
				}
				for _, step := range path.Steps {
					var nextNodes []*types.Node
					for _, n := range currentNodes {
						candidates := e.getChildNodes(n)

						var matchingNodes []*types.Node
						for _, child := range candidates {
							if child.Type == types.ElementNode && (step.Name == "*" || child.Name == step.Name) {
								matchingNodes = append(matchingNodes, child)
							}
						}

						currentStepNodes := matchingNodes
						for _, predicate := range step.Predicates {
							var filtered []*types.Node
							for i, m := range currentStepNodes {
								e.contextPosition = i + 1
								e.contextSize = len(currentStepNodes)
								if e.evaluateSimpleCondition(m, predicate) {
									filtered = append(filtered, m)
								}
							}
							currentStepNodes = filtered
						}
						nextNodes = append(nextNodes, currentStepNodes...)
					}
					currentNodes = nextNodes
				}
				return strconv.Itoa(len(currentNodes))
			}
			// For simple ElementExpression like count(li)
			if elem, ok := arg.(*ElementExpression); ok {
				count := 0
				for _, child := range node.Children {
					if child.Type == types.ElementNode && (elem.Name == "*" || child.Name == elem.Name) {
						count++
					}
				}
				return strconv.Itoa(count)
			}
			return "1"
		}
		return "0"

	case "position":
		return strconv.Itoa(e.contextPosition)

	case "last":
		return strconv.Itoa(e.contextSize)

	case "text":
		return node.TextContent

	case "true":
		return "true"

	case "false":
		return "false"

	default:
		Trace("unknown function: %s", fn.Name)
		return ""
	}
}
