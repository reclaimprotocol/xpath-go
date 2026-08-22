package utils

import (
	"testing"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

func TestSourceBoundaryKeepsEndLocationTogether(t *testing.T) {
	var boundary sourceBoundary
	boundary.set(12, 21, 3, 8)
	if boundary.contentEnd != 12 || boundary.end != (sourcePoint{pos: 21, line: 3, column: 8}) {
		t.Fatalf("unexpected boundary: %#v", boundary)
	}
}

func TestZeroOffsetBoundaryUsesUnsetSentinelSemantics(t *testing.T) {
	body := &types.Node{Type: types.ElementNode, Name: "body", StartLine: 1, StartColumn: 1}
	parser := &HTMLParser{
		htmlParserState: htmlParserState{
			documentState: documentState{documentBody: body},
			inputState:    inputState{pos: 5, line: 1, col: 6},
			templateState: templateState{templateOwner: make(map[*types.Node]*types.Node)},
		},
	}
	// A synthetic body boundary may have source offset zero. Keep the existing
	// sentinel behavior: it must not overwrite the normal EOF location unless
	// an explicit end tag established a positive end offset.
	parser.documentBodyBoundary = sourceBoundary{contentEnd: 0, end: sourcePoint{pos: 0, line: 1, column: 1}}
	parser.finishImplicitElement(body, "", 0)
	if body.EndPos != 5 || body.EndLine != 1 || body.EndColumn != 6 {
		t.Fatalf("zero-offset boundary was treated as closed: %#v", body)
	}
}

func TestAdoptionFlagsPreserveIndependentStateAndCleanUp(t *testing.T) {
	node := &types.Node{}
	parser := &HTMLParser{htmlParserState: htmlParserState{adoptionState: adoptionState{adoptionFlags: make(map[*types.Node]adoptionFlag)}}}
	parser.setAdoptionFlag(node, adoptionDetached)
	parser.setAdoptionFlag(node, adoptionClosed)
	if !parser.hasAdoptionFlag(node, adoptionDetached) || !parser.hasAdoptionFlag(node, adoptionClosed) {
		t.Fatalf("independent adoption flags were not retained: %#v", parser.adoptionFlags)
	}
	parser.clearAdoptionFlag(node, adoptionDetached)
	if parser.hasAdoptionFlag(node, adoptionDetached) || !parser.hasAdoptionFlag(node, adoptionClosed) {
		t.Fatalf("clearing one adoption flag changed another: %#v", parser.adoptionFlags)
	}
	parser.clearAdoptionFlag(node, adoptionClosed)
	if _, exists := parser.adoptionFlags[node]; exists {
		t.Fatalf("zero-value adoption state should be removed: %#v", parser.adoptionFlags)
	}
}

func TestTemplateOwnerCacheStoresKnownNilOwner(t *testing.T) {
	root := &types.Node{}
	node := &types.Node{Parent: root}
	parser := &HTMLParser{htmlParserState: htmlParserState{templateState: templateState{templateOwner: make(map[*types.Node]*types.Node)}}}
	if owner := parser.templateOwnerForNode(node); owner != nil {
		t.Fatalf("expected no template owner, got %#v", owner)
	}
	if owner, known := parser.templateOwner[node]; !known || owner != nil {
		t.Fatalf("missing cached nil template owner: %#v", parser.templateOwner)
	}
}

func TestTagPreviewCachesOnlySuccessfulTagNames(t *testing.T) {
	parser := &HTMLParser{htmlParserState: htmlParserState{inputState: inputState{content: "<DIV>", tagPreview: tagPreview{position: -1}}}}
	name, closing, ok := parser.peekTagName()
	if !ok || closing || name != "div" || parser.tagPreview.position != 0 {
		t.Fatalf("expected cached opening div preview, got %q %t %t %#v", name, closing, ok, parser.tagPreview)
	}

	invalid := &HTMLParser{htmlParserState: htmlParserState{inputState: inputState{content: "<", tagPreview: tagPreview{position: -1}}}}
	if _, _, ok := invalid.peekTagName(); ok || invalid.tagPreview.position != -1 {
		t.Fatalf("invalid preview should not be cached: %#v", invalid.tagPreview)
	}
}

func TestParserReuseResetsStateAndLocationCaches(t *testing.T) {
	parser := NewHTMLParser()
	first, err := parser.Parse(`<template><b><i>x`)
	if err != nil {
		t.Fatalf("first parse failed: %v", err)
	}
	oldNodes := make(map[*types.Node]struct{})
	var collect func(*types.Node)
	collect = func(node *types.Node) {
		if node == nil {
			return
		}
		oldNodes[node] = struct{}{}
		for _, child := range node.Children {
			collect(child)
		}
		collect(node.TemplateContent)
	}
	collect(first)
	previousFormatting := parser.openFormatting
	previousForeignEnds := parser.foreignIgnoredEnds
	if _, err := parser.Parse(`<p>ok</p>`); err != nil {
		t.Fatalf("reused parse failed: %v", err)
	}

	document, err := NewHTMLParser().Parse(`<p>ok</p>`)
	if err != nil {
		t.Fatalf("fresh parse failed: %v", err)
	}
	// A stale template owner, adoption flag, or source boundary would either
	// alter this ordinary paragraph's parent/range or leave per-document state
	// attached to nodes from the first parse.
	paragraphs := listElements(document, "p")
	if len(paragraphs) != 1 || paragraphs[0].TextContent != "ok" || paragraphs[0].StartPos != 0 || paragraphs[0].EndPos != 9 || paragraphs[0].ContentStart != 3 || paragraphs[0].ContentEnd != 5 {
		t.Fatalf("unexpected fresh parse paragraph: %#v", paragraphs)
	}
	for node := range parser.templateOwner {
		if _, exists := oldNodes[node]; exists {
			t.Fatalf("template owner cache retained node from previous Parse: %p", node)
		}
	}
	if len(parser.templateEOFBoundary) != 0 || len(parser.adoptionFlags) != 0 {
		t.Fatalf("per-document caches leaked across Parse: eof=%d adoption=%d", len(parser.templateEOFBoundary), len(parser.adoptionFlags))
	}
	if parser.activeTable != nil || parser.activeForm != nil || parser.activeLIStart != nil || parser.activeSelect != nil {
		t.Fatalf("domain active nodes leaked across Parse: table=%p form=%p li=%p select=%p", parser.activeTable, parser.activeForm, parser.activeLIStart, parser.activeSelect)
	}
	if len(parser.deferredTableFoster) != 0 || len(parser.openForms) != 0 || len(parser.closedForms) != 0 || len(parser.deferredFormSiblings) != 0 {
		t.Fatalf("domain recovery maps leaked across Parse: table=%d open=%d closed=%d deferred=%d", len(parser.deferredTableFoster), len(parser.openForms), len(parser.closedForms), len(parser.deferredFormSiblings))
	}
	if parser.listUnwindCache.position != -1 || parser.selectUnwindCache.position != -1 {
		t.Fatalf("unwind caches were not reset: list=%#v select=%#v", parser.listUnwindCache, parser.selectUnwindCache)
	}
	if parser.content != `<p>ok</p>` || parser.activeParagraph != nil || len(parser.openFormatting) != 0 || len(parser.foreignIgnoredEnds) != 0 {
		t.Fatalf("remaining parser state leaked across Parse: content=%q paragraph=%p formatting=%d foreign=%d", parser.content, parser.activeParagraph, len(parser.openFormatting), len(parser.foreignIgnoredEnds))
	}
	if !parser.quirksMode || parser.genericEndPos != -1 {
		t.Fatalf("document recovery state was not reset: quirks=%t generic=%d", parser.quirksMode, parser.genericEndPos)
	}
	previousFormatting[&types.Node{}] = true
	previousForeignEnds["stale"] = 1
	if len(parser.openFormatting) != 0 || len(parser.foreignIgnoredEnds) != 0 {
		t.Fatalf("Parse retained maps from a previous htmlParserState")
	}
}

func TestParserFacadePreservesConfigurationAcrossStateReplacement(t *testing.T) {
	parser := NewHTMLParser()
	parser.SetScriptingEnabled(true)
	firstState := parser.htmlParserState
	if _, err := parser.Parse(`<noscript><b>x</b></noscript>`); err != nil {
		t.Fatalf("first parse failed: %v", err)
	}
	if !parser.scriptingEnabled || parser.content != `<noscript><b>x</b></noscript>` {
		t.Fatalf("Parse changed facade configuration or did not install new state: enabled=%t state=%#v", parser.scriptingEnabled, parser.inputState)
	}
	if _, err := parser.Parse(`<p>next</p>`); err != nil {
		t.Fatalf("second parse failed: %v", err)
	}
	if !parser.scriptingEnabled || parser.content != `<p>next</p>` {
		t.Fatalf("state replacement changed parser configuration: enabled=%t content=%q", parser.scriptingEnabled, parser.content)
	}
	if firstState.content != "" || firstState.line != 0 || firstState.col != 0 {
		t.Fatalf("parser facade unexpectedly mutated a copied state: %#v", firstState.inputState)
	}
}

func TestDomainStateConstructorsInitializeRequiredMapsAndSentinels(t *testing.T) {
	document := newDocumentState()
	table := newTableState()
	forms := newFormState()
	lists := newListState()
	selects := newSelectState()
	if !document.beforeHTMLMode || document.documentTextStartOverride != -1 {
		t.Fatalf("unexpected document defaults: %#v", document)
	}
	if table.deferredTableFoster == nil || forms.openForms == nil || forms.closedForms == nil || forms.deferredFormSiblings == nil || forms.formEOFBoundaries == nil {
		t.Fatalf("domain constructor left a required recovery map nil")
	}
	if lists.listUnwindCache.position != -1 || selects.selectUnwindCache.position != -1 {
		t.Fatalf("domain constructor lost invalid-cache sentinels: list=%#v select=%#v", lists.listUnwindCache, selects.selectUnwindCache)
	}
}

func TestRemainingStateConstructorsInitializeRequiredMapsAndSentinels(t *testing.T) {
	input := newInputState("source")
	scope := newScopeState()
	formatting := newFormattingState()
	adoption := newAdoptionState()
	foreign := newForeignContentState()
	recovery := newRecoveryState()
	templates := newTemplateState()
	if input.content != "source" || input.line != 1 || input.col != 1 || input.tagPreview.position != -1 {
		t.Fatalf("unexpected input defaults: %#v", input)
	}
	if scope.activeParagraph != nil || scope.recoverOpenElementsAtEOF {
		t.Fatalf("unexpected scope defaults: %#v", scope)
	}
	if formatting.openFormatting == nil || formatting.formattingFamilies == nil || formatting.formattingByNode == nil || formatting.formattingText == nil {
		t.Fatalf("formatting constructor left a required map nil")
	}
	if adoption.adoptionSplit == nil || adoption.adoptionPendingEnds == nil || adoption.adoptionFlags == nil || adoption.adoptionIgnoredEnds == nil || adoption.adoptionUnwindPos != -1 || adoption.specialFormattingStartPos != -1 {
		t.Fatalf("unexpected adoption defaults: %#v", adoption)
	}
	if foreign.activeSVGNames == nil || foreign.foreignIgnoredEnds == nil || foreign.foreignIgnoredHTMLEnds == nil {
		t.Fatalf("foreign constructor left a required map nil")
	}
	if !recovery.quirksMode || recovery.genericEndPos != -1 {
		t.Fatalf("unexpected recovery defaults: %#v", recovery)
	}
	if templates.templateOwner == nil || templates.templateEOFBoundary == nil {
		t.Fatalf("template constructor left a required map nil")
	}
}

func TestTextBuilderPreservesDecodedValueAndSourceRange(t *testing.T) {
	const content = `<p>prefix &amp; é 😀 suffix</p>`
	document, err := NewHTMLParser().Parse(content)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	paragraphs := listElements(document, "p")
	if len(paragraphs) != 1 {
		t.Fatalf("expected one paragraph, got %d", len(paragraphs))
	}
	paragraph := paragraphs[0]
	if paragraph.TextContent != "prefix & é 😀 suffix" {
		t.Fatalf("decoded text mismatch: %q", paragraph.TextContent)
	}
	if paragraph.ContentStart != len(`<p>`) || paragraph.ContentEnd != len(content)-len(`</p>`) || paragraph.EndPos != len(content) {
		t.Fatalf("text builder changed source range: %#v", paragraph)
	}
}

func TestSelectUnwindCacheUsesTypedActions(t *testing.T) {
	cache := selectUnwindCache{position: 4, action: selectUnwindNestedSelect}
	if cache.action != selectUnwindNestedSelect {
		t.Fatalf("unexpected typed select action: %#v", cache.action)
	}
	if got := selectUnwindImplicit; got == selectUnwindEnd || got == selectUnwindNestedSelect {
		t.Fatalf("select action constants are not distinct: implicit=%d end=%d nested=%d", selectUnwindImplicit, selectUnwindEnd, selectUnwindNestedSelect)
	}
}
