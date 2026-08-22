package xpath

import "testing"

const benchmarkHTML = `<!doctype html><html><body><main><section id="items"><article data-kind="keep"><h2>First</h2><p>alpha</p><span>one</span></article><article data-kind="skip"><h2>Second</h2><p>beta</p><span>two</span></article><article data-kind="keep"><h2>Third</h2><p>gamma</p><span>three</span></article></section></main></body></html>`

// BenchmarkQuery measures the complete one-shot path: XPath parsing, HTML
// parsing, evaluation, and public result conversion.
func BenchmarkQuery(b *testing.B) {
	const expression = `//article[@data-kind='keep' and string(.//span)!=''][1]//span`
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkHTML)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := Query(expression, benchmarkHTML)
		if err != nil || len(results) == 0 {
			b.Fatalf("Query failed: %v (results=%d)", err, len(results))
		}
	}
}

// BenchmarkCompiledEvaluate is the companion benchmark for repeated use of a
// compiled query. It makes parser work visible in profiles and lets the
// refactor prove that compiled execution does not regress against Query.
func BenchmarkCompiledEvaluate(b *testing.B) {
	const expression = `//article[@data-kind='keep' and string(.//span)!=''][1]//span`
	compiled, err := Compile(expression)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(int64(len(benchmarkHTML)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := compiled.Evaluate(benchmarkHTML)
		if err != nil || len(results) == 0 {
			b.Fatalf("Evaluate failed: %v (results=%d)", err, len(results))
		}
	}
}

// BenchmarkPredicateHeavyEvaluation exercises nested predicates and typed
// conversions, which historically caused predicate source to be reparsed for
// every candidate node.
func BenchmarkPredicateHeavyEvaluation(b *testing.B) {
	const expression = `//article[number(@rank)=1 or (boolean(@enabled) and string(.//span)='one')][position()=1]`
	content := `<main>`
	for i := 0; i < 64; i++ {
		content += `<article rank="2" enabled="true"><span>other</span></article>`
	}
	content += `<article rank="1" enabled="true"><span>one</span></article></main>`
	b.ReportAllocs()
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := Query(expression, content)
		if err != nil || len(results) != 1 {
			b.Fatalf("predicate query failed: %v (results=%d)", err, len(results))
		}
	}
}

// BenchmarkResultPathConstruction keeps result conversion in the benchmark;
// path generation is intentionally exercised with repeated sibling names.
func BenchmarkResultPathConstruction(b *testing.B) {
	content := `<root>`
	for i := 0; i < 256; i++ {
		content += `<item><value>payload</value></item>`
	}
	content += `</root>`
	b.ReportAllocs()
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results, err := QueryWithOptions(`//item/value`, content, Options{OutputFormat: "paths"})
		if err != nil || len(results) != 256 {
			b.Fatalf("path query failed: %v (results=%d)", err, len(results))
		}
	}
}

func TestQueryAndCompiledEvaluateReturnEquivalentResults(t *testing.T) {
	const expression = `//article[@data-kind='keep']//span`
	queryResults, err := Query(expression, benchmarkHTML)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	compiled, err := Compile(expression)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}
	compiledResults, err := compiled.Evaluate(benchmarkHTML)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if len(queryResults) != len(compiledResults) {
		t.Fatalf("Query returned %d results, compiled evaluation returned %d", len(queryResults), len(compiledResults))
	}
	for i := range queryResults {
		if queryResults[i].Path != compiledResults[i].Path || queryResults[i].Value != compiledResults[i].Value {
			t.Fatalf("result %d differs: Query=%+v compiled=%+v", i, queryResults[i], compiledResults[i])
		}
	}
}

func TestCompileRejectsMalformedXPathAndUnknownConstructs(t *testing.T) {
	for _, expression := range []string{
		`//div[`,
		`//div[1 2]`,
		`//div/imaginary-axis::node()`,
		`//div[unknown-function()]`,
	} {
		if _, err := Compile(expression); err == nil {
			t.Errorf("Compile(%q) succeeded, want validation error", expression)
		}
	}
}
