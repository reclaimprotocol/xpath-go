package evaluator

import (
	"testing"
	"time"

	"github.com/reclaimprotocol/xpath-go/internal/parser"
	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func alternatingAxisDocument(pairs int) *types.Node {
	document := &types.Node{Type: types.DocumentNode, Name: "#document"}
	html := &types.Node{Type: types.ElementNode, Name: "html", Parent: document}
	head := &types.Node{Type: types.ElementNode, Name: "head", Parent: html}
	body := &types.Node{Type: types.ElementNode, Name: "body", Parent: html}
	main := &types.Node{Type: types.ElementNode, Name: "main", Parent: body}
	document.Children = []*types.Node{html}
	html.Children = []*types.Node{head, body}
	body.Children = []*types.Node{main}
	main.Children = make([]*types.Node, 0, 2*pairs)
	for i := 0; i < pairs; i++ {
		main.Children = append(main.Children,
			&types.Node{Type: types.ElementNode, Name: "b", Parent: main},
			&types.Node{Type: types.ElementNode, Name: "i", Parent: main},
		)
	}
	return document
}

func bestMultiContextAxisEvaluation(t *testing.T, document *types.Node, expression string, want int) time.Duration {
	t.Helper()
	parsed, err := parser.NewParser().Parse(expression)
	if err != nil {
		t.Fatal(err)
	}
	best := time.Duration(1<<63 - 1)
	for i := 0; i < 5; i++ {
		evaluator := NewEvaluator()
		start := time.Now()
		evaluator.prepareAxisOrder(document)
		results, evalErr := evaluator.evaluateSteps(parsed, document)
		elapsed := time.Since(start)
		if evalErr != nil || len(results) != want {
			t.Fatalf("multi-context axis %q: got %d results, want %d; err=%v", expression, len(results), want, evalErr)
		}
		if elapsed < best {
			best = elapsed
		}
	}
	return best
}

func TestFollowingAndPrecedingMultiContextScaling(t *testing.T) {
	const smallN, largeN = 1000, 4000
	smallDocument := alternatingAxisDocument(smallN)
	largeDocument := alternatingAxisDocument(largeN)
	for _, expression := range []string{
		`//b/following::i[1]`,
		`//i/preceding::b[1]`,
	} {
		// Warm the evaluator before taking best-of-five measurements. Building
		// the recovered DOM order is included; HTML tokenization is deliberately
		// excluded so this gate measures the multi-context axis algorithm itself.
		_ = bestMultiContextAxisEvaluation(t, smallDocument, expression, smallN)
		small := bestMultiContextAxisEvaluation(t, smallDocument, expression, smallN)
		large := bestMultiContextAxisEvaluation(t, largeDocument, expression, largeN)
		ratio := float64(large) / float64(small)
		t.Logf("multi-context axis %q: 1k=%v 4k=%v ratio=%.2fx", expression, small, large, ratio)
		if large > 8*small+5*time.Millisecond {
			t.Fatalf("multi-context axis %q scales superlinearly: 1k=%v 4k=%v ratio=%.2fx", expression, small, large, ratio)
		}
	}
}
