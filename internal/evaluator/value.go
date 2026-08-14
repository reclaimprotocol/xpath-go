package evaluator

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

type xpathValueKind uint8

const (
	invalidXPathValue xpathValueKind = iota
	nodeSetXPathValue
	stringXPathValue
	numberXPathValue
	booleanXPathValue
)

type xpathValue struct {
	kind    xpathValueKind
	text    string
	number  float64
	boolean bool
	nodes   []*types.Node
}

func stringValue(value string) xpathValue  { return xpathValue{kind: stringXPathValue, text: value} }
func numberValue(value float64) xpathValue { return xpathValue{kind: numberXPathValue, number: value} }
func booleanValue(value bool) xpathValue   { return xpathValue{kind: booleanXPathValue, boolean: value} }
func nodeSetValue(nodes ...*types.Node) xpathValue {
	return xpathValue{kind: nodeSetXPathValue, nodes: nodes}
}

func nodeStringValue(node *types.Node) string {
	if node == nil {
		return ""
	}
	switch node.Type {
	case types.AttributeNode, types.TextNode, types.CommentNode, types.ProcessingInstructionNode:
		if node.Value != "" || node.TextContent == "" {
			return node.Value
		}
		return node.TextContent
	case types.DocumentTypeNode:
		return ""
	default:
		return node.TextContent
	}
}

func (value xpathValue) toString() string {
	switch value.kind {
	case nodeSetXPathValue:
		if len(value.nodes) == 0 {
			return ""
		}
		return nodeStringValue(value.nodes[0])
	case stringXPathValue:
		return value.text
	case numberXPathValue:
		return formatXPathNumber(value.number)
	case booleanXPathValue:
		if value.boolean {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func (value xpathValue) toNumber() float64 {
	switch value.kind {
	case nodeSetXPathValue, stringXPathValue:
		return parseXPathNumber(value.toString())
	case numberXPathValue:
		return value.number
	case booleanXPathValue:
		if value.boolean {
			return 1
		}
		return 0
	default:
		return math.NaN()
	}
}

func (value xpathValue) toBoolean() bool {
	switch value.kind {
	case nodeSetXPathValue:
		return len(value.nodes) > 0
	case stringXPathValue:
		return value.text != ""
	case numberXPathValue:
		return value.number != 0 && !math.IsNaN(value.number)
	case booleanXPathValue:
		return value.boolean
	default:
		return false
	}
}

func (value xpathValue) legacyString() string {
	if value.kind == nodeSetXPathValue {
		if len(value.nodes) == 0 {
			return ""
		}
		return NodeSetPrefix + nodeStringValue(value.nodes[0])
	}
	return value.toString()
}

func parseXPathNumber(value string) float64 {
	value = strings.Trim(value, " \t\r\n")
	if value == "" {
		return math.NaN()
	}
	position := 0
	if value[position] == '-' {
		position++
		if position == len(value) {
			return math.NaN()
		}
	}
	digits := 0
	for position < len(value) && value[position] >= '0' && value[position] <= '9' {
		position++
		digits++
	}
	if position < len(value) && value[position] == '.' {
		position++
		for position < len(value) && value[position] >= '0' && value[position] <= '9' {
			position++
			digits++
		}
	}
	if digits == 0 || position != len(value) {
		return math.NaN()
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		if rangeErr, ok := err.(*strconv.NumError); ok && rangeErr.Err == strconv.ErrRange {
			return number
		}
		return math.NaN()
	}
	return number
}

func formatXPathNumber(value float64) string {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "Infinity"
	case math.IsInf(value, -1):
		return "-Infinity"
	case value == 0:
		return "0"
	default:
		return strconv.FormatFloat(value, 'f', -1, 64)
	}
}

func compareTypedXPathValues(left xpathValue, operator string, right xpathValue) bool {
	if operator == "=" || operator == "!=" {
		if left.kind == nodeSetXPathValue || right.kind == nodeSetXPathValue {
			return compareXPathNodeSets(left, operator, right)
		}
		switch {
		case left.kind == booleanXPathValue || right.kind == booleanXPathValue:
			return compareBooleans(left.toBoolean(), operator, right.toBoolean())
		case left.kind == numberXPathValue || right.kind == numberXPathValue:
			return compareNumbers(left.toNumber(), operator, right.toNumber())
		default:
			if operator == "=" {
				return left.toString() == right.toString()
			}
			return left.toString() != right.toString()
		}
	}

	if left.kind == nodeSetXPathValue || right.kind == nodeSetXPathValue {
		return compareXPathNodeSets(left, operator, right)
	}
	return compareNumbers(left.toNumber(), operator, right.toNumber())
}

func compareXPathNodeSets(left xpathValue, operator string, right xpathValue) bool {
	if left.kind == nodeSetXPathValue && right.kind == booleanXPathValue || right.kind == nodeSetXPathValue && left.kind == booleanXPathValue {
		return compareBooleans(left.toBoolean(), operator, right.toBoolean())
	}
	leftValues := scalarComparisonValues(left)
	rightValues := scalarComparisonValues(right)
	for _, leftValue := range leftValues {
		for _, rightValue := range rightValues {
			if operator == "=" || operator == "!=" {
				if left.kind == numberXPathValue || right.kind == numberXPathValue {
					if compareNumbers(leftValue.toNumber(), operator, rightValue.toNumber()) {
						return true
					}
				} else if operator == "=" && leftValue.toString() == rightValue.toString() || operator == "!=" && leftValue.toString() != rightValue.toString() {
					return true
				}
			} else if compareNumbers(leftValue.toNumber(), operator, rightValue.toNumber()) {
				return true
			}
		}
	}
	return false
}

func scalarComparisonValues(value xpathValue) []xpathValue {
	if value.kind != nodeSetXPathValue {
		return []xpathValue{value}
	}
	result := make([]xpathValue, 0, len(value.nodes))
	for _, node := range value.nodes {
		result = append(result, stringValue(nodeStringValue(node)))
	}
	return result
}

func compareBooleans(left bool, operator string, right bool) bool {
	if operator == "=" {
		return left == right
	}
	if operator == "!=" {
		return left != right
	}
	return compareNumbers(boolNumber(left), operator, boolNumber(right))
}

func boolNumber(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func compareNumbers(left float64, operator string, right float64) bool {
	switch operator {
	case "=":
		return left == right
	case "!=":
		return left != right
	case "<":
		return left < right
	case ">":
		return left > right
	case "<=":
		return left <= right
	case ">=":
		return left >= right
	default:
		return false
	}
}

func xpathStringLength(value string) int {
	return utf8.RuneCountInString(value)
}
