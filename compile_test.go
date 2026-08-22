package xpath

import (
	"sync"
	"testing"

	"github.com/reclaimprotocol/xpath-go/internal/evaluator"
	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestCompileValidatesWholeExpression(t *testing.T) {
	for _, expression := range []string{
		`//div[unknown()]`,
		`//namespace::item`,
		`//div[@id=]`,
		`//div |`,
	} {
		if compiled, err := Compile(expression); err == nil || compiled != nil {
			t.Errorf("Compile(%q) = %#v, %v; want validation error", expression, compiled, err)
		}
	}
}

func TestCompiledPredicateAndAbbreviatedSteps(t *testing.T) {
	compiled, err := Compile(`//span[@data-keep='yes']/..`)
	if err != nil {
		t.Fatal(err)
	}
	results, err := compiled.Evaluate(`<main><section><span data-keep="yes">x</span></section></main>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].NodeName != "section" {
		t.Fatalf("compiled parent-step results = %#v, want one section", results)
	}

	self, err := Query(`//span/.`, `<main><span>x</span></main>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(self) != 1 || self[0].NodeName != "span" {
		t.Fatalf("self-step results = %#v, want one span", self)
	}
}

func TestRootExpressionReturnsTheDocumentNode(t *testing.T) {
	results, err := Query(`/`, `<main>text</main>`)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].NodeType != int(types.DocumentNode) {
		t.Fatalf("root expression results = %#v, want one document node", results)
	}
	if _, err := Compile(`//`); err == nil {
		t.Fatal("Compile accepted incomplete descendant path")
	}
}

func TestCompiledEvaluationIsSafeWithTypedPredicates(t *testing.T) {
	compiled, err := Compile(`//li[@kind='keep' and position()=1]`)
	if err != nil {
		t.Fatal(err)
	}
	const workers, iterations = 8, 20
	errs := make(chan error, workers*iterations)
	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for iteration := 0; iteration < iterations; iteration++ {
				results, evalErr := compiled.Evaluate(`<ul><li kind="keep">one</li><li kind="keep">two</li></ul>`)
				if evalErr != nil || len(results) != 1 || results[0].TextContent != "one" {
					errs <- evalErr
				}
			}
		}()
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		} else {
			t.Error("compiled typed predicate returned an unexpected result")
		}
	}
}

func TestIncludeLocationFalseOmitsAllSourceOffsets(t *testing.T) {
	results, err := QueryWithOptions(`//p`, `<p>text</p>`, Options{IncludeLocation: false, OutputFormat: "nodes"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %#v, want one result", results)
	}
	result := results[0]
	if result.StartLocation != 0 || result.EndLocation != 0 || result.ContentStart != 0 || result.ContentEnd != 0 {
		t.Fatalf("IncludeLocation=false leaked source offsets: %#v", result)
	}
	if result.Value != `<p>text</p>` {
		t.Fatalf("IncludeLocation=false changed node value: %#v", result)
	}
}

func TestDebugOptionRestoresDisabledTraceState(t *testing.T) {
	DisableTrace()
	stop := enableEvaluationTrace(true)
	if !evaluator.IsTraceEnabled() {
		t.Fatal("debug evaluation did not enable tracing")
	}
	stop()
	if evaluator.IsTraceEnabled() {
		t.Fatal("debug evaluation left tracing enabled")
	}
}

func TestDebugTraceScopesOverlapSafely(t *testing.T) {
	DisableTrace()
	first := enableEvaluationTrace(true)
	second := enableEvaluationTrace(true)
	first()
	if !evaluator.IsTraceEnabled() {
		t.Fatal("first debug scope disabled a concurrent debug scope")
	}
	second()
	if evaluator.IsTraceEnabled() {
		t.Fatal("last debug scope did not restore disabled trace state")
	}
}
