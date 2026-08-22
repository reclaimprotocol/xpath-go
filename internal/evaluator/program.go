package evaluator

import (
	"fmt"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Program is an immutable, parsed XPath expression. It contains no
// document-specific state and can safely be evaluated concurrently by creating
// one Evaluator per call.
type Program struct {
	// root is the sole execution representation. It is immutable after Compile
	// and contains all paths, predicates and function calls as typed nodes.
	root Expression
}

// Compile parses an XPath expression and compiles every predicate into its
// typed expression tree. Parsing predicates here, instead of while filtering
// each candidate node, is both the execution boundary for invalid expressions
// and the critical hot-path optimization for predicate-heavy queries.
func Compile(expression string) (*Program, error) {
	root, err := parseXPath(expression)
	if err != nil {
		return nil, fmt.Errorf("XPath parsing failed: %w", err)
	}
	if !isNodeSetExpression(root) {
		return nil, fmt.Errorf("XPath parsing failed: top-level XPath expression must select nodes")
	}
	return &Program{root: root}, nil
}

// ParseLegacyXPath is a compatibility projection for the former
// internal/parser package. It delegates to the same lexer and typed parser as
// Compile, so there is no second XPath grammar in the repository.
func ParseLegacyXPath(expression string) (*types.ParsedXPath, error) {
	root, err := parseXPath(expression)
	if err != nil {
		return nil, err
	}
	if !isNodeSetExpression(root) {
		return nil, fmt.Errorf("top-level XPath expression must select nodes")
	}
	return compatibilityParsedXPath(root), nil
}

func isNodeSetExpression(expression Expression) bool {
	switch value := expression.(type) {
	case *PathExpression, *ElementExpression, *AxisExpression, *AttributeExpression:
		return true
	case *UnionExpression:
		for _, operand := range value.Operands {
			if !isNodeSetExpression(operand) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func compatibilityParsedXPath(expression Expression) *types.ParsedXPath {
	path, ok := expression.(*PathExpression)
	if ok {
		parsed := &types.ParsedXPath{IsAbsolute: path.IsAbsolute, Steps: make([]types.XPathStep, 0, len(path.Steps))}
		if path.IsDeep {
			parsed.Steps = append(parsed.Steps, types.XPathStep{Axis: types.AxisDescendantOrSelf, NodeTest: "node()"})
		}
		for _, step := range path.Steps {
			axis := types.AxisChild
			if step.Axis != "" {
				axis = types.XPathAxis(step.Axis)
			}
			predicate := make([]types.XPathPredicate, 0, len(step.Predicates))
			for _, condition := range step.Predicates {
				// Parsed intentionally remains nil: this is a data-only projection
				// for the deprecated parser package, never an execution bridge.
				predicate = append(predicate, types.XPathPredicate{Expression: condition.String()})
			}
			parsed.Steps = append(parsed.Steps, types.XPathStep{Axis: axis, NodeTest: step.Name, Predicates: predicate})
		}
		return parsed
	}
	if union, ok := expression.(*UnionExpression); ok {
		parsed := &types.ParsedXPath{Union: make([]*types.ParsedXPath, 0, len(union.Operands))}
		for _, operand := range union.Operands {
			parsed.Union = append(parsed.Union, compatibilityParsedXPath(operand))
		}
		return parsed
	}
	return &types.ParsedXPath{}
}
