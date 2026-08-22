// Package parser is a deprecated compatibility adapter. XPath syntax is
// implemented once in internal/evaluator's typed parser, which also compiles
// Program execution trees.
package parser

import (
	"fmt"
	"strings"

	"github.com/reclaimprotocol/xpath-go/internal/evaluator"
	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// Parser remains reusable for callers of the old internal package. It carries
// no lexer state; each call delegates to the shared typed parser.
type Parser struct{}

func NewParser() *Parser { return &Parser{} }

func (p *Parser) Parse(expression string) (*types.ParsedXPath, error) {
	if strings.TrimSpace(expression) == "" {
		return nil, fmt.Errorf("empty XPath expression")
	}
	parsed, err := evaluator.ParseLegacyXPath(expression)
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}
	adaptLegacySteps(parsed, topLevelPredicateSources(expression), new(int))
	return parsed, nil
}

func adaptLegacySteps(parsed *types.ParsedXPath, sources []string, next *int) {
	if parsed == nil {
		return
	}
	for index := range parsed.Steps {
		step := &parsed.Steps[index]
		switch step.NodeTest {
		case ".":
			step.Axis, step.NodeTest = types.AxisSelf, "node()"
		case "..":
			step.Axis, step.NodeTest = types.AxisParent, "node()"
		}
		for predicateIndex := range step.Predicates {
			if *next < len(sources) {
				step.Predicates[predicateIndex].Expression = sources[*next]
				*next++
			}
		}
	}
	for _, member := range parsed.Union {
		adaptLegacySteps(member, sources, next)
	}
}

// topLevelPredicateSources retains the exact legacy source string without
// participating in parsing. Nested brackets belong to expression AST nodes,
// not to the projected location path.
func topLevelPredicateSources(source string) []string {
	var values []string
	depth, start := 0, -1
	var quote byte
	for index := 0; index < len(source); index++ {
		if quote != 0 {
			if source[index] == quote {
				quote = 0
			}
			continue
		}
		if source[index] == '\'' || source[index] == '"' {
			quote = source[index]
			continue
		}
		switch source[index] {
		case '[':
			if depth == 0 {
				start = index + 1
			}
			depth++
		case ']':
			depth--
			if depth == 0 && start >= 0 {
				values = append(values, strings.TrimSpace(source[start:index]))
				start = -1
			}
		}
	}
	return values
}
