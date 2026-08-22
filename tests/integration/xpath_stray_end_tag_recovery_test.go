package xpath_test

import (
	"testing"

	xpath "github.com/reclaimprotocol/xpath-go"
)

// TestQueryStrayEndTagInsideTableIgnored covers the HTML5 "in table" / "in
// body" insertion-mode rules for an end tag whose name has no matching element
// in scope: "If the stack of open elements does not have an element in scope
// that is an HTML element with the same tag name as that of the token, then
// this is a parse error; ignore the token." Browsers drop a stray </div> that
// appears inside table content; the parser must not fail the whole document.
func TestQueryStrayEndTagInsideTableIgnored(t *testing.T) {
	const document = `<table><tr><td>x</td></tr></div></table>`

	results, err := xpath.Query(`//table/tbody/tr/td`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != "x" {
		t.Fatalf("Expected the stray end tag to be ignored and the cell to parse, got %#v", results)
	}
}

// TestQueryStrayEndTagOutsideTableScopeIgnored covers the same rule when the
// matching element is open above a table: the table is an ordinary-scope
// boundary, so the div is not in scope and the </div> token is ignored.
func TestQueryStrayEndTagOutsideTableScopeIgnored(t *testing.T) {
	const document = `<div><table><tr><td>x</td></tr></div></table>`

	results, err := xpath.Query(`//div/table/tbody/tr/td`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != "x" {
		t.Fatalf("Expected the </div> above the table to be ignored, got %#v", results)
	}
}

// TestQueryStrayEndTagInsideCellIgnored covers a stray end tag inside table
// cell content; the token is ignored and surrounding text is preserved.
func TestQueryStrayEndTagInsideCellIgnored(t *testing.T) {
	const document = `<table><tr><td>x</div>y</td></tr></table>`

	results, err := xpath.Query(`//table/tbody/tr/td`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != "xy" {
		t.Fatalf("Expected merged cell text 'xy', got %#v", results)
	}
}

// TestQueryPortalPageStrayEndTagRecoversNestedSpanText simulates a full-page
// portal layout (sidebar table carrying a stray </div>) that must still expose
// a nested span's text. jsdom reference: the stray </div> is ignored, the
// table is dropped back into the sidebar div, and the span text query returns
// the expected value.
func TestQueryPortalPageStrayEndTagRecoversNestedSpanText(t *testing.T) {
	const document = `<html><body><div class="wrapper">` +
		`<header class="main-header"><span class="logo-mini">V</span></header>` +
		`<aside class="main-sidebar"><div class="sidebar">` +
		`<table class="nav"><tr><td>Menu</td></tr></div></table>` +
		`</div></aside>` +
		`<div class="content-wrapper">` +
		`<div id="showBox" class="row">` +
		`<div class="card"><div class="card-header" data-background-color='pink'>` +
		`<i class="material-icons">person</i></div>` +
		`<div class="card-body"><span id="lblFullName">Student Name</span></div>` +
		`</div></div></div>` +
		`<footer class="main-footer"><strong>All rights reserved.</strong></footer>` +
		`</div></body></html>`

	results, err := xpath.Query(`//span[@id='lblFullName']/text()`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != "Student Name" {
		t.Fatalf("Expected the span text to be extracted, got %#v", results)
	}
}

// TestQueryStudentPortalStrayEndTagRecoversStudentName simulates a second
// production page pattern: an ASP.NET student portal (navigation header with
// a "Pay Online" link, menu table carrying a stray </div>) that must still
// expose a student-name span via //span[@id='stname']/text(). jsdom reference:
// the stray </div> is ignored and the span text query returns the name.
func TestQueryStudentPortalStrayEndTagRecoversStudentName(t *testing.T) {
	const document = `<html><body>` +
		`<form id="form1">` +
		`<div class="header"><div class="nav"><a href="payments.aspx">Pay Online</a></div></div>` +
		`<div class="sidebar"><table class="menu">` +
		`<tr><td>Home</td></tr><tr><td>Results</td></tr></div></table></div>` +
		`<div class="content"><div class="profile">` +
		`<span id="stname">Student Name</span>` +
		`<span id="senroll">12345</span>` +
		`</div></div>` +
		`</form></body></html>`

	results, err := xpath.Query(`//span[@id='stname']/text()`, document)
	if err != nil {
		t.Fatalf("Query returned an error: %v", err)
	}
	if len(results) != 1 || results[0].TextContent != "Student Name" {
		t.Fatalf("Expected the student name span to be extracted, got %#v", results)
	}
}
