package evaluator

import "testing"

func finalPathPredicate(t *testing.T, program *Program) Expression {
	t.Helper()
	path, ok := program.root.(*PathExpression)
	if !ok || len(path.Steps) == 0 || len(path.Steps[len(path.Steps)-1].Predicates) == 0 {
		t.Fatalf("compiled program has unexpected typed shape: %#v", program.root)
	}
	return path.Steps[len(path.Steps)-1].Predicates[0]
}

func TestCompileBuildsTypedPredicatesOnce(t *testing.T) {
	program, err := Compile(`//item[@rank='primary' and position()=1]`)
	if err != nil {
		t.Fatal(err)
	}
	predicate := finalPathPredicate(t, program)
	if _, ok := predicate.(*BooleanExpression); !ok {
		t.Fatalf("compiled predicate = %T, want typed boolean expression", predicate)
	}

	evaluator := NewEvaluator()
	results, err := evaluator.EvaluateProgramWithDocument(program, `<item rank="primary">first</item><item rank="primary">second</item>`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].TextContent != "first" {
		t.Fatalf("compiled predicate results = %#v", results)
	}
}

func TestEvaluateProgramDoesNotReparseCompiledExpression(t *testing.T) {
	program, err := Compile(`//item[@rank='primary' and position()=1]`)
	if err != nil {
		t.Fatal(err)
	}
	compiledPredicate := finalPathPredicate(t, program)
	compiledRoot := program.root

	// EvaluateProgramWithDocument accepts only the immutable Program, not an
	// expression string. Keep the typed predicate identity across repeated
	// evaluations: any per-candidate or per-evaluation parser fallback would
	// either replace this value or leave it nil. This is an architecture gate
	// for the compiled-evaluation zero-expression-parsing contract.
	e := NewEvaluator()
	for i, content := range []string{
		`<item rank="primary">first</item><item rank="secondary">skip</item>`,
		`<item rank="primary">second</item><item rank="secondary">skip</item>`,
		`<item rank="primary">third</item>`,
	} {
		results, evalErr := e.EvaluateProgramWithDocument(program, content, nil)
		if evalErr != nil {
			t.Fatalf("evaluation %d failed: %v", i, evalErr)
		}
		if len(results) != 1 {
			t.Fatalf("evaluation %d returned %d results, want one", i, len(results))
		}
		predicate := finalPathPredicate(t, program)
		if predicate != compiledPredicate {
			t.Fatalf("evaluation %d replaced compiled predicate (%p -> %p)", i, compiledPredicate, predicate)
		}
		if program.root != compiledRoot {
			t.Fatalf("evaluation %d replaced immutable compiled root", i)
		}
	}
}

func TestCompiledPredicateSupportsRelativeDescendantPaths(t *testing.T) {
	program, err := Compile(`//article[string(.//span)='keep']`)
	if err != nil {
		t.Fatal(err)
	}
	evaluator := NewEvaluator()
	results, err := evaluator.EvaluateProgramWithDocument(program, `<article><span>keep</span></article><article><span>skip</span></article>`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].TextContent != "keep" {
		t.Fatalf("relative descendant predicate results = %#v", results)
	}
}

func TestUnifiedParserCompilesNestedPathPredicates(t *testing.T) {
	expression, err := parseXPath(`count(.//span[@rank='keep'])`)
	if err != nil {
		t.Fatal(err)
	}
	function := expression.(*FunctionExpression)
	path, ok := function.Function.Arguments[0].(*PathExpression)
	if !ok || len(path.Steps) != 2 || len(path.Steps[1].Predicates) != 1 {
		t.Fatalf("nested path was not compiled: %#v", expression)
	}
	if _, ok := path.Steps[1].Predicates[0].(*ComparisonExpression); !ok {
		t.Fatalf("nested predicate type = %T, want *ComparisonExpression", path.Steps[1].Predicates[0])
	}

	program, err := Compile(`//article[count(.//span[@rank='keep'])=1]`)
	if err != nil {
		t.Fatal(err)
	}
	results, err := NewEvaluator().EvaluateProgramWithDocument(program, `<article><span rank="keep">x</span></article><article><span rank="skip">y</span></article>`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].TextContent != "x" {
		t.Fatalf("nested path predicate results = %#v", results)
	}
}
