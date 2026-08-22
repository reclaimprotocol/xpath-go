// Parser configuration, per-parse state domains, source positions, and
// strongly typed insertion/recovery modes.
package utils

import (
	"strings"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// HTMLParser parses HTML/XML with location tracking
type HTMLParser struct {
	// scriptingEnabled is parser configuration, not per-Parse state. The zero
	// value follows the scripting-disabled tree builder used by default Query.
	scriptingEnabled bool
	htmlParserState
}

// htmlParserState contains all mutable state for one Parse call. It is
// embedded to preserve the parser's direct field access while keeping the
// HTMLParser facade limited to configuration plus replaceable parse state.
type htmlParserState struct {
	inputState
	scopeState
	documentState
	tableState
	formState
	listState
	selectState
	formattingState
	adoptionState
	foreignContentState
	recoveryState
	templateState
	textContentState
}

func newHTMLParserState(content string) htmlParserState {
	return htmlParserState{
		inputState:          newInputState(content),
		scopeState:          newScopeState(),
		documentState:       newDocumentState(),
		tableState:          newTableState(),
		formState:           newFormState(),
		listState:           newListState(),
		selectState:         newSelectState(),
		formattingState:     newFormattingState(),
		adoptionState:       newAdoptionState(),
		foreignContentState: newForeignContentState(),
		recoveryState:       newRecoveryState(),
		templateState:       newTemplateState(),
		textContentState:    newTextContentState(),
	}
}

// textContentState records subtrees whose aggregate TextContent may be stale
// after recovery surgery. Normal parsing builds text incrementally; only
// reparenting and child-list edits enter this invalidation path.
type textContentState struct {
	dirtyTextNodes map[*types.Node]struct{}
	refreshVisits  int
}

func newTextContentState() textContentState {
	return textContentState{dirtyTextNodes: make(map[*types.Node]struct{})}
}

type formattingEntry struct {
	template *types.Node
	current  *types.Node
	family   string
	index    int
	marker   bool
}

// inputState owns the source cursor and token preview cache. The parser keeps
// byte offsets and UTF-16 coordinates together so speculative parser copies
// retain a complete tokenizer view.
type inputState struct {
	content          string
	pos              int
	line             int
	col              int
	lastWasCR        bool
	utf8Continuation int
	tagPreview       tagPreview
}

func newInputState(content string) inputState {
	return inputState{content: content, line: 1, col: 1, tagPreview: tagPreview{position: -1}}
}

// scopeState contains open-element recovery that is independent of a
// particular insertion mode.
type scopeState struct {
	// openElements is the tree-builder's live stack of open elements. It is
	// maintained at the recursive parser boundary, so recovery algorithms can
	// inspect the current insertion context without rebuilding it from Parent
	// links on every token.
	openElements               []*types.Node
	recoverOpenElementsAtEOF   bool
	preserveNextRootWhitespace bool
	bodyContentStarted         bool
	headDepth                  int
	headExited                 bool
	activeParagraph            *types.Node
	activeParagraphAncestors   map[string]struct{}
	activeButton               *types.Node
	paragraphAncestorClose     string
	nodesClosedAtDocumentEnd   []*types.Node
}

func newScopeState() scopeState { return scopeState{} }

// pushOpenElement records one recursive tree-builder frame. The stack is
// deliberately package-private: callers still use the existing recovery
// helpers, while parser state now has one explicit owner for open-element
// movement.
func (p *HTMLParser) pushOpenElement(node *types.Node) {
	if node != nil {
		p.openElements = append(p.openElements, node)
	}
}

func (p *HTMLParser) popOpenElement(node *types.Node) {
	if node == nil || len(p.openElements) == 0 {
		return
	}
	last := len(p.openElements) - 1
	if p.openElements[last] == node {
		p.openElements = p.openElements[:last]
		return
	}
	// Recovery can return through a frame after an adoption agency move. Keep
	// the stack usable even when that frame is no longer the physical suffix.
	for index := last; index >= 0; index-- {
		if p.openElements[index] == node {
			copy(p.openElements[index:], p.openElements[index+1:])
			p.openElements = p.openElements[:last]
			return
		}
	}
}

// appendChildIncremental attaches a node and updates the parent's direct text
// aggregate in the same operation. Tree-builder recovery can still request a
// full rebuild after moving an existing subtree, but ordinary append paths no
// longer need to rescan the parent's descendants just to incorporate one new
// child.
func appendChildIncremental(parent, child *types.Node) {
	if parent == nil || child == nil {
		return
	}
	child.Parent = parent
	if child.Type == types.TextNode && len(parent.Children) > 0 && parent.Children[len(parent.Children)-1].Type == types.TextNode {
		mergeTextNodes(parent.Children[len(parent.Children)-1], child)
	} else {
		parent.Children = append(parent.Children, child)
	}
	if parent.Type == types.DocumentNode {
		return
	}
	// Small containers benefit from an immediately available aggregate. For a
	// long sibling stream, repeated string concatenation is quadratic; the
	// parser's dirty finalization boundary rebuilds that aggregate once instead.
	if len(parent.Children) > 32 {
		return
	}
	switch child.Type {
	case types.TextNode:
		parent.TextContent += child.Value
	case types.ElementNode:
		parent.TextContent += child.TextContent
	}
}

type formattingState struct {
	activeFormatting           []*formattingEntry
	openFormatting             map[*types.Node]bool
	formattingOpenStack        []*types.Node
	formattingContainer        map[*types.Node]*types.Node
	formattingTable            map[*types.Node]*types.Node
	formattingFamilies         map[string][]*formattingEntry
	formattingByName           map[string][]*formattingEntry
	formattingByNode           map[*types.Node]*formattingEntry
	formattingText             map[*types.Node]*strings.Builder
	elementContentStart        map[*types.Node]sourcePoint
	formattingMarkerStart      int
	formattingDocumentEnd      bool
	formattingDocumentEndPrior bool
}

func newFormattingState() formattingState {
	return formattingState{
		openFormatting:      make(map[*types.Node]bool),
		formattingContainer: make(map[*types.Node]*types.Node),
		formattingTable:     make(map[*types.Node]*types.Node),
		formattingFamilies:  make(map[string][]*formattingEntry),
		formattingByName:    make(map[string][]*formattingEntry),
		formattingByNode:    make(map[*types.Node]*formattingEntry),
		formattingText:      make(map[*types.Node]*strings.Builder),
		elementContentStart: make(map[*types.Node]sourcePoint),
	}
}

type adoptionState struct {
	adoptionSplit              map[*types.Node][]*types.Node
	adoptionPendingEnds        map[*types.Node][]*types.Node
	adoptionFlags              map[*types.Node]adoptionFlag
	adoptionEOFBoundary        *types.Node
	adoptionUnwindPos          int
	adoptionUnwindFormatting   *types.Node
	adoptionUnwindBlock        *types.Node
	adoptionUnwindInScope      bool
	adoptionIgnoredEnds        map[string]int
	specialFormattingStartPos  int
	specialFormattingStartName string
}

func newAdoptionState() adoptionState {
	return adoptionState{
		adoptionSplit:             make(map[*types.Node][]*types.Node),
		adoptionPendingEnds:       make(map[*types.Node][]*types.Node),
		adoptionFlags:             make(map[*types.Node]adoptionFlag),
		adoptionUnwindPos:         -1,
		adoptionIgnoredEnds:       make(map[string]int),
		specialFormattingStartPos: -1,
	}
}

type foreignContentState struct {
	activeSVGNames          map[string]*types.Node
	foreignIgnoredEnds      map[string]int
	foreignIgnoredHTMLEnds  map[string]int
	foreignIgnoredEndCount  int
	foreignHTMLDepth        int
	foreignBreakoutRecovery bool
	foreignBreakoutBoundary *types.Node
	foreignBRReprocess      bool
}

func newForeignContentState() foreignContentState {
	return foreignContentState{
		activeSVGNames:         make(map[string]*types.Node),
		foreignIgnoredEnds:     make(map[string]int),
		foreignIgnoredHTMLEnds: make(map[string]int),
	}
}

type recoveryState struct {
	quirksMode       bool
	genericEndPos    int
	genericEndName   string
	genericEndTarget *types.Node
}

func newRecoveryState() recoveryState {
	return recoveryState{quirksMode: true, genericEndPos: -1}
}

type templateState struct {
	activeTemplates           []*types.Node
	templateInsertionModes    []templateInsertionMode
	templateForeignTransition bool
	templateOwner             map[*types.Node]*types.Node
	templateEOFBoundary       map[*types.Node]sourcePoint
}

func newTemplateState() templateState {
	return templateState{
		templateOwner:       make(map[*types.Node]*types.Node),
		templateEOFBoundary: make(map[*types.Node]sourcePoint),
	}
}

// documentState owns the implicit-document insertion modes and source-backed
// wrapper locations. It is embedded so existing parser algorithms can keep
// their direct field access while the state has one reset boundary.
type documentState struct {
	documentHTML              *types.Node
	documentHead              *types.Node
	documentBody              *types.Node
	documentFrameset          *types.Node
	documentRoot              *types.Node
	beforeHTMLMode            bool
	explicitDocumentSkeleton  bool
	explicitAfterAfterBody    bool
	documentTextStartOverride int
	documentTextStartLine     int
	documentTextStartColumn   int
	documentEndRecovery       bool
	documentBodyBoundary      sourceBoundary
	documentHTMLBoundary      sourceBoundary
	// documentMode is the currently dispatched explicit-document insertion
	// mode. Keeping it in parse state makes mode movement observable and
	// replaceable instead of hiding it solely in a local switch variable.
	documentMode explicitDocumentMode
}

func newDocumentState() documentState {
	return documentState{beforeHTMLMode: true, documentTextStartOverride: -1}
}

// tableState contains table insertion-mode and foster-parenting state.
type tableState struct {
	activeTable         *types.Node
	deferredTableFoster map[*types.Node][]*types.Node
	tableFosterDepth    tableInsertionDepth
	tableFosterSection  string
	tableFosterRow      bool
	tableCellDepth      tableInsertionDepth
	tableCellName       string
	tableCellSection    string
	tableCaptionDepth   tableInsertionDepth
}

func newTableState() tableState {
	return tableState{deferredTableFoster: make(map[*types.Node][]*types.Node)}
}

// formState keeps the form pointer and recovery state together.
type formState struct {
	activeForm           *types.Node
	openForms            map[*types.Node]bool
	closedForms          map[*types.Node]bool
	deferredFormSiblings map[*types.Node][]*types.Node
	formEOFBoundaries    map[*types.Node]bool
}

func newFormState() formState {
	return formState{
		openForms:            make(map[*types.Node]bool),
		closedForms:          make(map[*types.Node]bool),
		deferredFormSiblings: make(map[*types.Node][]*types.Node),
		formEOFBoundaries:    make(map[*types.Node]bool),
	}
}

// listState is the list-item insertion state and its document-end recovery.
type listState struct {
	activeLIStart           *types.Node
	activeDefinitionStart   *types.Node
	activeLIEnd             *types.Node
	activeDDEnd             *types.Node
	activeDTEnd             *types.Node
	listEOFRecoveryBoundary *types.Node
	listDocumentEndMode     listDocumentEndMode
	deferredHTMLComments    []*types.Node
	deferredRootComments    []*types.Node
	listUnwindCache         listUnwindCache
}

// tableInsertionDepth tracks nested tree-builder reprocessing contexts. Its
// named zero value is the inactive insertion mode; a depth is required because
// a reprocessed token can enter the same context recursively.
type tableInsertionDepth uint8

const tableInsertionInactive tableInsertionDepth = 0

func (depth tableInsertionDepth) active() bool { return depth != tableInsertionInactive }

// listDocumentEndMode identifies the document-end insertion mode without
// relying on string spellings in recovery paths.
type listDocumentEndMode uint8

const (
	listDocumentEndNone listDocumentEndMode = iota
	listDocumentEndBody
	listDocumentEndHTML
)

func newListState() listState {
	return listState{listUnwindCache: listUnwindCache{position: -1}}
}

// selectState contains select/option scope and cached unwind decisions.
type selectState struct {
	activeSelect              *types.Node
	activeOptionStart         *types.Node
	activeOptgroupStart       *types.Node
	activeOptionEnd           *types.Node
	activeOptgroupEnd         *types.Node
	activeSelectImpliedEnd    *types.Node
	activeSelectOptionImplied *types.Node
	activeSelectNames         map[string]*types.Node
	activeSelectAllNames      map[string]*types.Node
	selectUnwindCache         selectUnwindCache
	selectDocumentEndRecovery *types.Node
	selectBlockedByTable      bool
}

func newSelectState() selectState {
	return selectState{selectUnwindCache: selectUnwindCache{position: -1}}
}

// sourcePoint is a byte position together with its source coordinates.
// Keeping them together prevents partially updated parser locations.
type sourcePoint struct {
	pos    int
	line   int
	column int
}

// sourceBoundary records the source boundary before an explicit end tag and
// the location immediately after that tag.
type sourceBoundary struct {
	contentEnd int
	end        sourcePoint
}

func (b *sourceBoundary) set(contentEnd, endPos, endLine, endColumn int) {
	b.contentEnd = contentEnd
	b.end = sourcePoint{pos: endPos, line: endLine, column: endColumn}
}

type listUnwindCache struct {
	position int
	target   *types.Node
	explicit bool
}

type selectUnwindAction uint8

const (
	selectUnwindImplicit selectUnwindAction = iota
	selectUnwindEnd
	selectUnwindNestedSelect
)

type selectUnwindCache struct {
	position int
	target   *types.Node
	action   selectUnwindAction
}

type adoptionFlag uint8

const (
	adoptionDetached adoptionFlag = 1 << iota
	adoptionClosed
	adoptionReturnAtSplit
)

type tagPreview struct {
	position      int
	name          string
	closing       bool
	emits         bool
	emissionKnown bool
}

func (p *HTMLParser) hasAdoptionFlag(node *types.Node, flag adoptionFlag) bool {
	return p.adoptionFlags[node]&flag != 0
}

func (p *HTMLParser) setAdoptionFlag(node *types.Node, flag adoptionFlag) {
	p.adoptionFlags[node] |= flag
}

func (p *HTMLParser) clearAdoptionFlag(node *types.Node, flag adoptionFlag) {
	flags := p.adoptionFlags[node] &^ flag
	if flags == 0 {
		delete(p.adoptionFlags, node)
		return
	}
	p.adoptionFlags[node] = flags
}
