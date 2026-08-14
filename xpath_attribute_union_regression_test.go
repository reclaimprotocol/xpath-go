package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

func TestQueryAttributeUnionUsesDocumentOrderAndDeduplicates(t *testing.T) {
	const ordered = `<div a=1 b=2></div>`
	results, err := xpath.Query(`//@b | //@a`, ordered)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].NodeName != "a" || results[0].Value != "1" || results[1].NodeName != "b" || results[1].Value != "2" {
		t.Fatalf("Attribute union must return owner attribute order [a,b], got %#v", results)
	}

	const duplicate = `<div a=1></div>`
	results, err = xpath.Query(`//@a | //@a`, duplicate)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].NodeName != "a" || results[0].Value != "1" {
		t.Fatalf("Attribute union must deduplicate the same owner attribute, got %#v", results)
	}
}
