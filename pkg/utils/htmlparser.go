package utils

import (
	"fmt"
	stdhtml "html"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/reclaimprotocol/xpath-go/pkg/types"
)

// HTMLParser parses HTML/XML with location tracking
type HTMLParser struct {
	// scriptingEnabled is parser configuration, not per-Parse state. The zero
	// value follows the scripting-disabled tree builder used by default Query.
	scriptingEnabled bool
	content          string
	pos              int
	line             int
	col              int
	lastWasCR        bool
	// utf8Continuation tracks source bytes already counted as one UTF-16
	// column by advance. This keeps coordinates incremental and O(1).
	utf8Continuation int
	// recoverOpenElementsAtEOF is set only when the tokenizer reaches EOF in
	// an end-tag-related state. It propagates browser EOF closure through the
	// recursive open-element stack without making all unterminated HTML valid.
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
	documentHTML               *types.Node
	documentHead               *types.Node
	documentBody               *types.Node
	documentFrameset           *types.Node
	documentRoot               *types.Node
	beforeHTMLMode             bool
	explicitDocumentSkeleton   bool
	explicitAfterAfterBody     bool
	documentTextStartOverride  int
	documentTextStartLine      int
	documentTextStartColumn    int
	documentEndRecovery        bool
	activeTable                *types.Node
	activeForm                 *types.Node
	openForms                  map[*types.Node]bool
	closedForms                map[*types.Node]bool
	deferredFormSiblings       map[*types.Node][]*types.Node
	formEOFBoundaries          map[*types.Node]bool
	deferredTableFoster        map[*types.Node][]*types.Node
	tableFosterMode            int
	tableFosterSection         string
	tableFosterRow             bool
	tableCellMode              int
	tableCellName              string
	tableCellSection           string
	tableCaptionMode           int
	activeLIStart              *types.Node
	activeDefinitionStart      *types.Node
	activeLIEnd                *types.Node
	activeDDEnd                *types.Node
	activeDTEnd                *types.Node
	listEOFRecoveryBoundary    *types.Node
	documentBodyEndPos         int
	documentBodyContentEnd     int
	documentBodyEndLine        int
	documentBodyEndColumn      int
	documentHTMLEndPos         int
	documentHTMLContentEnd     int
	documentHTMLEndLine        int
	documentHTMLEndColumn      int
	listDocumentEndMode        string
	deferredHTMLComments       []*types.Node
	deferredRootComments       []*types.Node
	listUnwindCachePos         int
	listUnwindCacheTarget      *types.Node
	listUnwindCacheExplicit    bool
	activeSelect               *types.Node
	activeOptionStart          *types.Node
	activeOptgroupStart        *types.Node
	activeOptionEnd            *types.Node
	activeOptgroupEnd          *types.Node
	activeSelectImpliedEnd     *types.Node
	activeSelectOptionImplied  *types.Node
	activeSelectNames          map[string]*types.Node
	activeSelectAllNames       map[string]*types.Node
	selectUnwindPos            int
	selectUnwindTarget         *types.Node
	selectUnwindAction         string
	selectDocumentEndRecovery  *types.Node
	selectBlockedByTable       bool
	activeFormatting           []*formattingEntry
	openFormatting             map[*types.Node]bool
	formattingOpenStack        []*types.Node
	formattingContainer        map[*types.Node]*types.Node
	formattingTable            map[*types.Node]*types.Node
	formattingCounts           map[string]int
	formattingFamilies         map[string][]*formattingEntry
	formattingByName           map[string][]*formattingEntry
	formattingByNode           map[*types.Node]*formattingEntry
	formattingText             map[*types.Node]*strings.Builder
	elementContentStartLine    map[*types.Node]int
	elementContentStartColumn  map[*types.Node]int
	formattingMarkerStart      int
	formattingDocumentEnd      bool
	formattingDocumentEndPrior bool
	adoptionSplit              map[*types.Node][]*types.Node
	adoptionDetached           map[*types.Node]bool
	adoptionClosed             map[*types.Node]bool
	adoptionPendingEnds        map[*types.Node][]*types.Node
	adoptionReturnAtSplit      map[*types.Node]bool
	adoptionEOFBoundary        *types.Node
	adoptionUnwindPos          int
	adoptionUnwindFormatting   *types.Node
	adoptionUnwindBlock        *types.Node
	adoptionUnwindInScope      bool
	adoptionIgnoredEnds        map[string]int
	specialFormattingStartPos  int
	specialFormattingStartName string
	tagPreviewPos              int
	tagPreviewName             string
	tagPreviewClosing          bool
	tagPreviewOK               bool
	tagPreviewEmits            bool
	tagPreviewEmissionKnown    bool
	activeSVGNames             map[string]*types.Node
	foreignIgnoredEnds         map[string]int
	foreignIgnoredHTMLEnds     map[string]int
	foreignIgnoredEndCount     int
	foreignHTMLDepth           int
	foreignBreakoutRecovery    bool
	foreignBreakoutBoundary    *types.Node
	foreignBRReprocess         bool
	quirksMode                 bool
	genericEndPos              int
	genericEndName             string
	genericEndTarget           *types.Node
	activeTemplates            []*types.Node
	templateInsertionModes     []templateInsertionMode
	templateForeignTransition  bool
	templateOwner              map[*types.Node]*types.Node
	templateOwnerKnown         map[*types.Node]bool
	templateEOFBoundary        map[*types.Node]int
	templateEOFBoundaryLine    map[*types.Node]int
	templateEOFBoundaryColumn  map[*types.Node]int
}

type formattingEntry struct {
	template *types.Node
	current  *types.Node
	family   string
	index    int
	marker   bool
}

// NewHTMLParser creates a parser that reports malformed HTML/XML.
func NewHTMLParser() *HTMLParser {
	return &HTMLParser{}
}

// SetScriptingEnabled configures whole-document noscript parsing. When true,
// noscript contents use RAWTEXT; when false, markup is parsed using the
// scripting-disabled body and in-head-noscript rules.
func (p *HTMLParser) SetScriptingEnabled(enabled bool) {
	p.scriptingEnabled = enabled
}

// Parse parses HTML/XML content into a node tree with location information
func (p *HTMLParser) Parse(content string) (*types.Node, error) {
	if isBinaryInput(content) {
		return nil, fmt.Errorf("binary input is not supported")
	}

	p.content = content
	p.pos = 0
	p.line = 1
	p.col = 1
	p.lastWasCR = false
	p.utf8Continuation = 0
	p.recoverOpenElementsAtEOF = false
	p.preserveNextRootWhitespace = false
	p.bodyContentStarted = false
	p.headDepth = 0
	p.headExited = false
	p.activeParagraph = nil
	p.activeParagraphAncestors = nil
	p.activeButton = nil
	p.paragraphAncestorClose = ""
	p.nodesClosedAtDocumentEnd = nil
	p.documentHTML = nil
	p.documentHead = nil
	p.documentBody = nil
	p.documentFrameset = nil
	p.documentRoot = nil
	p.beforeHTMLMode = true
	p.explicitDocumentSkeleton = false
	p.explicitAfterAfterBody = false
	p.documentTextStartOverride = -1
	p.documentTextStartLine = 0
	p.documentTextStartColumn = 0
	p.documentEndRecovery = false
	p.activeTable = nil
	p.activeForm = nil
	p.openForms = make(map[*types.Node]bool)
	p.closedForms = make(map[*types.Node]bool)
	p.deferredFormSiblings = make(map[*types.Node][]*types.Node)
	p.formEOFBoundaries = make(map[*types.Node]bool)
	p.deferredTableFoster = make(map[*types.Node][]*types.Node)
	p.tableFosterMode = 0
	p.tableFosterSection = ""
	p.tableFosterRow = false
	p.tableCellMode = 0
	p.tableCellName = ""
	p.tableCellSection = ""
	p.tableCaptionMode = 0
	p.activeLIStart = nil
	p.activeDefinitionStart = nil
	p.activeLIEnd = nil
	p.activeDDEnd = nil
	p.activeDTEnd = nil
	p.listEOFRecoveryBoundary = nil
	p.documentBodyEndPos = 0
	p.documentBodyContentEnd = 0
	p.documentBodyEndLine = 0
	p.documentBodyEndColumn = 0
	p.documentHTMLEndPos = 0
	p.documentHTMLContentEnd = 0
	p.documentHTMLEndLine = 0
	p.documentHTMLEndColumn = 0
	p.listDocumentEndMode = ""
	p.deferredHTMLComments = nil
	p.deferredRootComments = nil
	p.listUnwindCachePos = -1
	p.listUnwindCacheTarget = nil
	p.listUnwindCacheExplicit = false
	p.activeSelect = nil
	p.activeOptionStart = nil
	p.activeOptgroupStart = nil
	p.activeOptionEnd = nil
	p.activeOptgroupEnd = nil
	p.activeSelectImpliedEnd = nil
	p.activeSelectOptionImplied = nil
	p.activeSelectNames = nil
	p.activeSelectAllNames = nil
	p.selectUnwindPos = -1
	p.selectUnwindTarget = nil
	p.selectUnwindAction = ""
	p.selectDocumentEndRecovery = nil
	p.selectBlockedByTable = false
	p.activeFormatting = nil
	p.openFormatting = make(map[*types.Node]bool)
	p.formattingOpenStack = nil
	p.formattingContainer = make(map[*types.Node]*types.Node)
	p.formattingTable = make(map[*types.Node]*types.Node)
	p.formattingCounts = make(map[string]int)
	p.formattingFamilies = make(map[string][]*formattingEntry)
	p.formattingByName = make(map[string][]*formattingEntry)
	p.formattingByNode = make(map[*types.Node]*formattingEntry)
	p.formattingText = make(map[*types.Node]*strings.Builder)
	p.elementContentStartLine = make(map[*types.Node]int)
	p.elementContentStartColumn = make(map[*types.Node]int)
	p.formattingMarkerStart = 0
	p.formattingDocumentEnd = false
	p.formattingDocumentEndPrior = false
	p.adoptionSplit = make(map[*types.Node][]*types.Node)
	p.adoptionDetached = make(map[*types.Node]bool)
	p.adoptionClosed = make(map[*types.Node]bool)
	p.adoptionPendingEnds = make(map[*types.Node][]*types.Node)
	p.adoptionReturnAtSplit = make(map[*types.Node]bool)
	p.adoptionEOFBoundary = nil
	p.adoptionUnwindPos = -1
	p.adoptionUnwindFormatting = nil
	p.adoptionUnwindBlock = nil
	p.adoptionUnwindInScope = false
	p.adoptionIgnoredEnds = make(map[string]int)
	p.specialFormattingStartPos = -1
	p.specialFormattingStartName = ""
	p.tagPreviewPos = -1
	p.tagPreviewEmissionKnown = false
	p.activeSVGNames = make(map[string]*types.Node)
	p.foreignIgnoredEnds = make(map[string]int)
	p.foreignIgnoredHTMLEnds = make(map[string]int)
	p.foreignIgnoredEndCount = 0
	p.foreignHTMLDepth = 0
	p.foreignBreakoutRecovery = false
	p.foreignBreakoutBoundary = nil
	p.foreignBRReprocess = false
	p.quirksMode = true
	p.genericEndPos = -1
	p.genericEndName = ""
	p.genericEndTarget = nil
	p.activeTemplates = nil
	p.templateInsertionModes = nil
	p.templateForeignTransition = false
	p.templateOwner = make(map[*types.Node]*types.Node)
	p.templateOwnerKnown = make(map[*types.Node]bool)
	p.templateEOFBoundary = make(map[*types.Node]int)
	p.templateEOFBoundaryLine = make(map[*types.Node]int)
	p.templateEOFBoundaryColumn = make(map[*types.Node]int)

	// Create document root
	root := &types.Node{
		Type:        types.DocumentNode,
		Name:        "#document",
		Children:    []*types.Node{},
		StartPos:    0,
		EndPos:      len(content),
		StartLine:   1,
		StartColumn: 1,
	}
	p.documentRoot = root

	// Parse child nodes
	for p.pos < len(content) {
		if len(p.adoptionSplit) > 0 {
			p.appendAllDeferredAdoption(root)
		}
		pendingTarget := p.pendingDocumentEndTarget()
		pendingName, pendingClosing, pendingTag := p.peekTagName()
		blockAfterDocumentEnd := pendingTarget != nil && pendingTag && !pendingClosing && p.paragraphClosingStartTag(pendingName) && p.currentTagTokenWillEmit()
		if blockAfterDocumentEnd {
			pendingTarget = p.resolvePendingThroughParagraph()
		} else if pendingTarget != nil && pendingTag && !pendingClosing && pendingName == "table" && p.quirksMode && p.currentTagTokenWillEmit() {
			for _, pending := range p.nodesClosedAtDocumentEnd {
				if pending.Name == "p" {
					pendingTarget = pending
					break
				}
			}
		}
		if pendingTarget != nil {
			if name, closing, ok := p.peekTagName(); ok && closing {
				if resolved, err := p.resolvePendingThroughName(name, true); err != nil {
					return nil, err
				} else if resolved {
					// Resolving a still-open descendant after </html> re-enters
					// in-body processing. Subsequent comments therefore belong to
					// BODY, not the after-after-body Document position.
					if p.explicitAfterAfterBody && name != "html" && name != "body" {
						p.explicitAfterAfterBody = false
					}
					continue
				}
			}
			if pendingTarget == nil && len(p.nodesClosedAtDocumentEnd) > 0 {
				pendingTarget = p.pendingDocumentEndTarget()
			}
			if pendingTarget == nil && pendingTag && pendingClosing && (pendingName == "p" || pendingName == "button") {
				p.skipWhitespace()
				if p.pos < len(content) {
					if name, closing, ok := p.peekTagName(); !ok || !closing || name != pendingName {
						pendingTarget = p.pendingDocumentEndTarget()
					}
				}
			}
		}
		if !p.preserveNextRootWhitespace && pendingTarget == nil && !p.documentEndRecovery && len(p.activeFormatting) == 0 {
			p.skipWhitespace()
		}
		p.preserveNextRootWhitespace = false
		if p.pos >= len(content) {
			break
		}
		// Every whole-document parse uses the browser document insertion
		// modes.  Keep preamble comments, processing instructions, and the
		// accepted doctype as Document children, but create a locationless
		// HTML element before reprocessing the first token that exits the
		// before-html mode.  This must happen before parsing that token: doing
		// a final tree wrap would lose the real insertion parent used by table
		// foster parenting, active formatting, forms, lists, and foreign
		// content.
		if p.beforeHTMLMode && p.documentHTML == nil {
			if whitespace := p.parseLeadingDocumentWhitespace(root); whitespace != nil {
				// Character tokens that are whitespace are ignored in the
				// before-html insertion mode (including decoded references).
				continue
			}
			if p.pos >= len(content) {
				break
			}
			name, closing, tag := p.peekTagName()
			qualifyingHTML := tag && !closing && name == "html" && p.currentTagTokenWillEmit()
			preambleToken := p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() || p.nextTokenIsIgnoredDoctype() || p.isMissingEndTagName()
			ignoredEnd := tag && closing && name != "head" && name != "body" && name != "html" && name != "br"
			if !qualifyingHTML && !preambleToken && !ignoredEnd {
				html := syntheticDocumentWrapper("html", root)
				p.documentHTML = html
				p.explicitDocumentSkeleton = true
				p.beforeHTMLMode = false
				parsed, err := p.parseExplicitDocumentHTML(html, 0)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(root, parsed)
				continue
			}
		}
		// In the before-html insertion mode, unrelated end tags are parse
		// errors that are ignored and do not force creation of the document
		// element. Keeping this state intact lets a later emitted <html> opt
		// into the explicit-document skeleton path.
		if p.beforeHTMLMode {
			if name, closing, ok := p.peekTagName(); ok && closing && name != "head" && name != "body" && name != "html" && name != "br" {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				continue
			}
		}
		if name, closing, ok := p.peekTagName(); ok && closing && p.adoptionIgnoredEnds[name] > 0 && p.activeFormattingEntryByName(name) == nil {
			start := p.pos
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			p.extendPreviousTextRange(root, start)
			p.adoptionIgnoredEnds[name]--
			continue
		}
		if name, closing, ok := p.peekTagName(); ok && closing && p.foreignIgnoredEnds[name] > 0 {
			start := p.pos
			if _, emitted, err := p.parseClosingTag(); err != nil {
				return nil, err
			} else if !emitted {
				p.extendPreviousTextRange(root, start)
				p.recoverOpenElementsAtEOF = true
			}
			p.retireForeignIgnoredEnd(name)
			continue
		}
		if name, closing, ok := p.peekTagName(); ok && closing && p.foreignIgnoredHTMLEnds[name] > 0 {
			start := p.pos
			if _, emitted, err := p.parseClosingTag(); err != nil {
				return nil, err
			} else if !emitted {
				p.extendPreviousTextRange(root, start)
			}
			p.foreignIgnoredHTMLEnds[name]--
			continue
		}
		if handled, err := p.handleForeignDocumentStart(); handled || err != nil {
			if err != nil {
				return nil, err
			}
			continue
		}
		if handled, err := p.handleFormToken(root, false); handled || err != nil {
			if err != nil {
				return nil, err
			}
			continue
		}
		if name, closing, ok := p.peekTagName(); ok && closing && isFormattingElement(name) {
			if isCoreAdoptionSubject(name) {
				if entry := p.activeFormattingEntryByName(name); entry == nil || entry.current == nil {
					start := p.pos
					if _, _, err := p.parseClosingTag(); err != nil {
						return nil, err
					}
					if entry != nil {
						p.removeActiveFormattingEntry(entry)
					}
					p.extendPreviousTextRange(root, start)
					continue
				}
			}
			if entry := p.activeFormattingEntryByName(name); entry != nil {
				if !p.openFormatting[entry.current] {
					start := p.pos
					if _, _, err := p.parseClosingTag(); err != nil {
						return nil, err
					}
					p.extendPreviousTextRange(root, start)
					p.removeActiveFormattingEntry(entry)
					continue
				}
				tokenStart := p.pos
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				p.refreshFormattingChain(entry.current)
				entry.current.ContentEnd = tokenStart
				entry.current.EndPos = p.pos
				entry.current.EndLine = p.line
				entry.current.EndColumn = p.col
				p.removeActiveFormattingEntry(entry)
				continue
			}
		}
		// At the document/body insertion level, an absent generic end tag is a
		// parse error that the HTML tree builder ignores. Paragraph and br ends
		// have dedicated in-body recovery below; all formatting/form/list/select
		// subjects have already had their identity-aware handlers above.
		if name, closing, ok := p.peekTagName(); ok && closing && name != "p" && name != "br" &&
			!p.isBogusEndTagOpen() && !p.isMissingEndTagName() && p.currentTagTokenWillEmit() {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		parent := root
		processingInstruction := p.nextTokenIsProcessingInstruction()
		commentToken := p.nextTokenIsComment()
		attachToPending := pendingTarget != nil && !p.nextTokenIsComment() && !processingInstruction
		attachToBody := p.documentEndRecovery && pendingTarget == nil && p.documentBody != nil &&
			(!p.explicitAfterAfterBody || (!commentToken && !processingInstruction))
		if attachToBody && p.explicitAfterAfterBody && !commentToken && !processingInstruction {
			p.explicitAfterAfterBody = false
		}
		if attachToPending {
			parent = pendingTarget
		} else if attachToBody {
			parent = p.documentBody
		}
		if action, err := p.handleSpecialFormattingStart(parent, "", 0); err != nil {
			return nil, err
		} else if action == adoptionContinue {
			continue
		}
		insertionParent := parent
		if !processingInstruction {
			insertionParent = p.formattingInsertionParent(parent, p.nextTokenReconstructsFormatting())
		}
		node, err := p.parseNode(insertionParent)
		if err != nil {
			return nil, err
		}
		if node != nil {
			if insertionParent != parent {
				p.appendFormattingChild(parent, insertionParent, node)
				if p.isSelfClosingTag(node.Name) {
					continue
				}
				// Non-void child parsing already consumed the following content.
				continue
			}
			if attachToPending {
				p.appendToPendingDocumentEnd(pendingTarget, node)
				continue
			}
			if attachToBody {
				p.appendToDocumentBody(node)
				continue
			}
			if node.Type == types.TextNode && !isHTMLWhitespaceString(node.Value) {
				p.bodyContentStarted = true
				p.beforeHTMLMode = false
			}
			if node.Type == types.ElementNode && (node.Name != "html" || !p.explicitDocumentSkeleton) {
				p.beforeHTMLMode = false
			}
			// The initial insertion phase ignores whitespace character tokens.
			// Test the decoded DOM value as well as the source bytes so numeric
			// references such as "&#32;" behave like literal leading spaces.
			if node.Type == types.TextNode && !p.doctypeWasIgnoredAfterBodyContent(root) && isHTMLWhitespaceString(node.Value) {
				continue
			}
			if node.Type == types.TextNode && len(root.Children) > 0 && root.Children[len(root.Children)-1].Type == types.TextNode {
				mergeTextNodes(root.Children[len(root.Children)-1], node)
			} else {
				root.Children = append(root.Children, node)
			}
			p.appendDeferredTableFoster(root, node)
			p.appendDeferredListComments(root, node)
			p.appendDeferredAdoption(root, node)
			p.appendDeferredFormSiblings(root, node)
		}
	}
	// EOF in the before-html insertion mode still creates the complete
	// locationless document skeleton. This covers empty, whitespace-only,
	// comment-only, and doctype-only documents.
	if p.documentHTML == nil {
		html := syntheticDocumentWrapper("html", root)
		p.documentHTML = html
		p.explicitDocumentSkeleton = true
		p.beforeHTMLMode = false
		parsed, err := p.parseExplicitDocumentHTML(html, 0)
		if err != nil {
			return nil, err
		}
		appendDocumentChild(root, parsed)
	}

	// Calculate final position
	if len(p.adoptionSplit) > 0 {
		p.appendAllDeferredAdoption(root)
	}
	p.refreshOpenFormattingText()
	for _, child := range root.Children {
		if child.Type == types.ElementNode {
			refreshTreeText(child)
		}
	}
	if p.documentBody != nil {
		refreshNodeTextContent(p.documentBody)
	}
	if p.documentHTML != nil {
		refreshNodeTextContent(p.documentHTML)
	}
	removeMisplacedDoctypes(root)
	for _, node := range p.nodesClosedAtDocumentEnd {
		node.EndPos = p.pos
		node.ContentEnd = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
	}
	normalizeSyntheticLocations(root)
	root.EndLine = p.line
	root.EndColumn = p.col
	root.SourceLength = len(content)

	return root, nil
}

func normalizeSyntheticLocations(node *types.Node) {
	for _, child := range node.Children {
		if child.Type != types.ElementNode {
			continue
		}
		if child.NamespaceURI == "" {
			child.NamespaceURI = htmlElementNamespace
		}
		normalizeSyntheticLocations(child)
		if child.StartLine == 0 {
			child.StartPos, child.ContentStart, child.ContentEnd, child.EndPos = 0, 0, 0, 0
			child.StartColumn, child.EndLine, child.EndColumn = 0, 0, 0
		}
	}
}

func removeMisplacedDoctypes(root *types.Node) {
	seenDoctype, seenElement := false, false
	children := root.Children[:0]
	for _, child := range root.Children {
		switch child.Type {
		case types.DocumentTypeNode:
			if seenDoctype || seenElement {
				continue
			}
			seenDoctype = true
		case types.ElementNode, types.TextNode:
			seenElement = true
		}
		children = append(children, child)
	}
	root.Children = children
}

func (p *HTMLParser) nextTokenIsComment() bool {
	if p.pos >= len(p.content) || p.peek() != '<' {
		return false
	}
	remaining := p.content[p.pos:]
	return strings.HasPrefix(remaining, "<!--") || strings.HasPrefix(remaining, "<!") ||
		p.isBogusEndTagOpen()
}

func (p *HTMLParser) nextTokenIsProcessingInstruction() bool {
	return p.pos+1 < len(p.content) && p.content[p.pos] == '<' && p.content[p.pos+1] == '?'
}

func (p *HTMLParser) pendingDocumentEndTarget() *types.Node {
	if len(p.nodesClosedAtDocumentEnd) == 0 {
		return nil
	}
	return p.nodesClosedAtDocumentEnd[0]
}

func (p *HTMLParser) pendingDocumentEndBlockStarts() bool {
	name, closing, ok := p.peekTagName()
	return ok && !closing && p.paragraphClosingStartTag(name) && p.currentTagTokenWillEmit()
}

func (p *HTMLParser) resolvePendingDocumentEndNodes() {
	for _, node := range p.nodesClosedAtDocumentEnd {
		node.EndPos = p.pos
		node.ContentEnd = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
	}
	p.nodesClosedAtDocumentEnd = nil
}

func (p *HTMLParser) resolvePendingThroughParagraph() *types.Node {
	for index, node := range p.nodesClosedAtDocumentEnd {
		if node.Name == "p" {
			p.resolvePendingPrefix(index + 1)
			return p.pendingDocumentEndTarget()
		}
	}
	p.resolvePendingDocumentEndNodes()
	return nil
}

func (p *HTMLParser) resolvePendingThroughName(name string, consume bool) (bool, error) {
	for index, node := range p.nodesClosedAtDocumentEnd {
		if node.Name != name {
			continue
		}
		if consume {
			tokenStart := p.pos
			tokenStartLine, tokenStartColumn := p.line, p.col
			_, emitted, err := p.parseClosingTag()
			if err != nil {
				return false, err
			}
			if !emitted {
				return false, nil
			}
			for pendingIndex, pending := range p.nodesClosedAtDocumentEnd[:index+1] {
				end := tokenStart
				if pendingIndex == index {
					end = p.pos
				}
				pending.EndPos = end
				pending.ContentEnd = tokenStart
				if pendingIndex == index {
					pending.EndLine = p.line
					pending.EndColumn = p.col
				} else {
					pending.EndLine = tokenStartLine
					pending.EndColumn = tokenStartColumn
				}
			}
			p.nodesClosedAtDocumentEnd = p.nodesClosedAtDocumentEnd[index+1:]
			return true, nil
		}
		p.resolvePendingPrefix(index + 1)
		return true, nil
	}
	return false, nil
}

func (p *HTMLParser) resolvePendingPrefix(count int) {
	if count > len(p.nodesClosedAtDocumentEnd) {
		count = len(p.nodesClosedAtDocumentEnd)
	}
	for _, node := range p.nodesClosedAtDocumentEnd[:count] {
		node.EndPos = p.pos
		node.ContentEnd = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
	}
	p.nodesClosedAtDocumentEnd = p.nodesClosedAtDocumentEnd[count:]
}

func (p *HTMLParser) appendToPendingDocumentEnd(target, node *types.Node) {
	if target == nil {
		return
	}
	node.Parent = target
	if node.Type == types.TextNode && len(target.Children) > 0 && target.Children[len(target.Children)-1].Type == types.TextNode {
		mergeTextNodes(target.Children[len(target.Children)-1], node)
	} else {
		target.Children = append(target.Children, node)
	}
	value := node.TextContent
	switch node.Type {
	case types.TextNode:
		value = node.Value
	case types.CommentNode, types.ProcessingInstructionNode:
		value = ""
	}
	for _, pending := range p.nodesClosedAtDocumentEnd {
		pending.TextContent += value
		pending.EndPos = p.pos
		pending.ContentEnd = p.pos
		pending.EndLine = p.line
		pending.EndColumn = p.col
	}
	p.addDocumentTextContent(value)
}

func (p *HTMLParser) addDocumentTextContent(value string) {
	if p.documentBody != nil {
		p.documentBody.TextContent += value
	}
	if p.documentHTML != nil {
		p.documentHTML.TextContent += value
	}
}

func (p *HTMLParser) appendToDocumentBody(node *types.Node) {
	if p.documentBody == nil {
		return
	}
	node.Parent = p.documentBody
	p.documentBody.Children = append(p.documentBody.Children, node)
	value := node.TextContent
	switch node.Type {
	case types.TextNode:
		value = node.Value
	case types.CommentNode, types.ProcessingInstructionNode:
		value = ""
	}
	p.addDocumentTextContent(value)
}

func isBinaryInput(content string) bool {
	if strings.HasPrefix(content, "\x1f\x8b") || !utf8.ValidString(content) {
		return true
	}

	for i := 0; i < len(content); i++ {
		c := content[i]
		if c != 0 && ((c < ' ' && c != '\t' && c != '\n' && c != '\r' && c != '\f') || c == 0x7f) {
			return true
		}
	}
	return false
}

// parseNode parses a single node (element, text, comment, etc.)
func (p *HTMLParser) parseNode(parent *types.Node) (*types.Node, error) {
	startPos := p.pos
	startLine := p.line
	startCol := p.col

	if p.peek() == '<' && p.startsMarkupToken() {
		if p.isBogusEndTagOpen() {
			return p.parseBogusEndTagComment(parent, startPos, startLine, startCol)
		}
		if p.isMissingEndTagName() {
			p.advanceRune(3) // Ignore "</>".
			if p.doctypeWasIgnoredAfterBodyContent(parent) {
				p.preserveNextRootWhitespace = true
			}
			return nil, nil
		}
		if recovered, consumed, err := p.consumeVoidElementClosingTag(parent); consumed || err != nil {
			return recovered, err
		}
		if p.pos+2 < len(p.content) && p.content[p.pos:p.pos+2] == "</" && isASCIIAlpha(p.content[p.pos+2]) {
			tokenStart := p.pos
			name, emitted, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if !emitted {
				p.extendPreviousTextRange(parent, tokenStart)
				return nil, nil
			}
			// The "in body" tree builder ignores a script end tag when
			// there is no script element in scope. This commonly follows a
			// correctly closed script after escaped script-data recovery.
			if name == "script" {
				return nil, nil
			}
			if name == "li" || name == "dd" || name == "dt" {
				return nil, nil
			}
			if name == "select" && (p.activeSelect == nil || p.selectBlockedByTable) {
				return nil, nil
			}
			if name == "p" {
				if parent.Type == types.DocumentNode && !p.bodyContentStarted {
					return nil, nil
				}
				return syntheticParagraph(parent), nil
			}
			return nil, fmt.Errorf("unexpected closing tag </%s> at position %d", name, tokenStart)
		}
		return p.parseElement(parent, startPos, startLine, startCol)
	}

	// Parse text node
	return p.parseTextNode(parent, startPos, startLine, startCol)
}

// parseElement parses an HTML/XML element
func (p *HTMLParser) parseElement(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	if p.peek() != '<' {
		return nil, fmt.Errorf("expected '<' at position %d", p.pos)
	}

	// Check for special elements
	if p.pos+1 < len(p.content) {
		if p.content[p.pos+1] == '!' {
			if strings.HasPrefix(p.content[p.pos:], "<!--") {
				return p.parseComment(parent, startPos, startLine, startCol)
			}
			if strings.HasPrefix(strings.ToUpper(p.content[p.pos:]), "<!DOCTYPE") {
				doctype, err := p.parseDoctype(parent, startPos, startLine, startCol)
				if err != nil {
					return nil, err
				}
				if p.shouldKeepDoctype(parent) {
					return doctype, nil
				}
				if p.doctypeWasIgnoredAfterBodyContent(parent) {
					p.preserveNextRootWhitespace = true
				}
				return nil, nil
			}
			return p.parseBogusComment(parent, startPos, startLine, startCol)
		}
		if p.content[p.pos+1] == '?' {
			return p.parseProcessingInstruction(parent, startPos, startLine, startCol)
		}
	}

	p.advance() // Skip '<'

	// Check for closing tag
	if p.peek() == '/' {
		return nil, fmt.Errorf("unexpected closing tag at position %d", startPos)
	}

	// The tag-open state only creates a start tag when the first character is
	// ASCII alpha. Other less-than sequences are emitted by parseTextNode.
	tagName := p.parseTagName()
	if tagName == "" {
		return nil, fmt.Errorf("expected tag name at position %d", p.pos)
	}
	// The in-body tree builder reparses the legacy image start tag as img.
	if tagName == "image" {
		tagName = "img"
	}

	node := &types.Node{
		Type:           types.ElementNode,
		Name:           tagName,
		NamespaceURI:   htmlElementNamespace,
		Attributes:     make(map[string]string),
		AttributeOrder: []string{},
		Children:       []*types.Node{},
		Parent:         parent,
		StartPos:       startPos,
		StartLine:      startLine,
		StartColumn:    startCol,
	}
	if owner := p.templateOwner[parent]; owner != nil {
		p.templateOwner[node] = owner
	}

	selfClosing, emitted := p.parseStartTagTail(node)
	if !emitted {
		// EOF in any start-tag state discards the incomplete token. parse5 and
		// browsers associate its original bytes with an immediately preceding
		// text node even though those bytes do not contribute to its value.
		p.extendPreviousTextRange(parent, startPos)
		return nil, nil
	}
	node.ContentStart = p.pos
	if selfClosing && p.foreignBreakoutRecovery && !p.isSelfClosingTag(node.Name) {
		// HTML ignores the self-closing flag on an ordinary element. parse5's
		// recovered content boundary begins at the flag's solidus.
		node.ContentStart = p.pos - 2
	}
	switch node.Name {
	case "svg":
		node.NamespaceURI = svgElementNamespace
		adjustSVGAttributes(node)
		initializeAttributeMetadata(node, true)
	case "math":
		node.NamespaceURI = mathElementNamespace
		adjustMathMLAttributes(node)
		initializeAttributeMetadata(node, true)
	default:
		initializeAttributeMetadata(node, false)
	}
	if node.NamespaceURI == htmlElementNamespace && p.foreignIgnoredEnds[node.Name] > 0 {
		// A real HTML element with the same name supersedes an omitted stale
		// foreign close. Its eventual end tag belongs to this new identity.
		p.retireForeignIgnoredEnd(node.Name)
	}
	if node.NamespaceURI == htmlElementNamespace && p.foreignBreakoutRecovery && p.foreignBreakoutBoundary == nil && !p.isSelfClosingTag(node.Name) {
		p.foreignBreakoutBoundary = node
		defer func() {
			if p.foreignBreakoutBoundary == node {
				p.foreignBreakoutBoundary = nil
				p.foreignBreakoutRecovery = p.foreignIgnoredEndCount > 0
			}
		}()
	}
	qualifyingDocumentHTML := node.Name == "html" && parent == p.documentRoot && p.beforeHTMLMode && p.documentHTML == nil
	if qualifyingDocumentHTML {
		p.explicitDocumentSkeleton = true
		p.beforeHTMLMode = false
	}
	if node.Name == "form" {
		previousForm := p.activeForm
		insideTemplate := p.templateOwnerForNode(parent) != nil
		p.activeForm = node
		p.openForms[node] = true
		p.formEOFBoundaries[node] = true
		if insideTemplate {
			defer func() { p.activeForm = previousForm }()
		}
	}
	p.elementContentStartLine[node] = p.line
	p.elementContentStartColumn[node] = p.col
	if isFormattingElement(node.Name) {
		p.addActiveFormatting(node)
		p.setFormattingOpen(node, true)
		defer p.setFormattingOpen(node, false)
	}
	restoreElementContext := p.enterElementContext(node)
	defer restoreElementContext()
	if !isFormattingElement(node.Name) {
		defer p.closeFormattingOnElementReturn(node)
	}
	if qualifyingDocumentHTML {
		return p.parseExplicitDocumentHTML(node, node.ContentStart)
	}
	if p.explicitDocumentSkeleton && node.Name == "head" && node.Parent == p.documentHTML {
		return p.parseExplicitDocumentHead(node, node.ContentStart)
	}
	if node.Name == "template" && node.NamespaceURI == htmlElementNamespace {
		return p.parseTemplateElement(node, node.ContentStart)
	}
	if node.Name == "noscript" && !p.scriptingEnabled && node.Parent == p.documentHead {
		return p.parseHeadNoscriptDisabled(node, node.ContentStart)
	}

	// In HTML, a self-closing flag on an ordinary element is acknowledged as
	// a parse error but does not close the element. Void elements remain empty.
	if p.isSelfClosingTag(node.Name) {
		node.EndPos = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
		// For self-closing tags, content start and end are the same (no inner content)
		node.ContentStart = p.pos
		node.ContentEnd = p.pos
		for _, entry := range p.activeFormatting {
			current := entry.current
			if entry.marker || current == nil || !p.openFormatting[current] || !nodeWithin(parent, current) {
				continue
			}
			current.TextContent += node.TextContent
			current.ContentEnd = p.pos
			current.EndPos = p.pos
			current.EndLine = p.line
			current.EndColumn = p.col
		}
		return node, nil
	}
	if node.NamespaceURI == svgElementNamespace || node.NamespaceURI == mathElementNamespace {
		if selfClosing {
			p.finishForeignSelfClosing(node)
			return node, nil
		}
		if p.foreignHTMLDepth > 0 {
			previousNames := p.activeSVGNames
			p.activeSVGNames = make(map[string]*types.Node)
			defer func() { p.activeSVGNames = previousNames }()
		}
		return p.parseSVGElement(node, p.pos)
	}

	// Mark the start of inner content (after opening tag)
	contentStartPos := node.ContentStart
	if node.Name == "frameset" && p.explicitDocumentSkeleton && p.documentBody == nil &&
		(parent == p.documentHTML || (parent != nil && parent.Name == "frameset")) {
		return p.parseDocumentFramesetElement(node, contentStartPos)
	}
	if node.Name == "table" {
		return p.parseTableElement(node, contentStartPos)
	}

	// Handle raw text elements like script, style, textarea, title
	if p.isRawTextElement(node.Name) || (node.Name == "noscript" && p.scriptingEnabled) {
		textState, err := p.parseRawTextContentWithPos(node.Name)
		if err != nil {
			return nil, err
		}
		textContent := textState.content

		// Create a single text node for the raw content
		if textContent != "" {
			textNode := &types.Node{
				Type:        types.TextNode,
				Name:        "#text",
				Value:       textContent,
				TextContent: textContent,
				Parent:      node,
				StartPos:    textState.start,
				EndPos:      textState.end,
				StartLine:   textState.startLine,
				StartColumn: textState.startColumn,
				EndLine:     textState.endLine,
				EndColumn:   textState.endColumn,
			}
			node.Children = append(node.Children, textNode)
		}

		node.TextContent = textContent
		node.ContentStart = contentStartPos
		node.ContentEnd = textState.end
		node.EndPos = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
		return node, nil
	}

	if node.Name == "plaintext" {
		textState := p.parsePlaintextContent()
		if textState.content != "" {
			textParent := p.formattingInsertionParent(node, true)
			textNode := &types.Node{
				Type: types.TextNode, Name: "#text", Value: textState.content, TextContent: textState.content,
				Parent: textParent, StartPos: textState.start, EndPos: textState.end,
				StartLine: textState.startLine, StartColumn: textState.startColumn,
				EndLine: textState.endLine, EndColumn: textState.endColumn,
			}
			if textParent != node {
				p.appendFormattingChild(node, textParent, textNode)
			} else {
				textParent.Children = append(textParent.Children, textNode)
			}
		}
		node.TextContent = textState.content
		node.ContentStart = contentStartPos
		node.ContentEnd = textState.end
		node.EndPos = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
		return node, nil
	}

	// Parse child nodes normally for other elements
	var textContent strings.Builder
	suppressInitialPreformattedLF := node.NamespaceURI == htmlElementNamespace && (node.Name == "pre" || node.Name == "listing")
	preformattedRangeStart := -1
	preformattedRangeLine, preformattedRangeColumn := 0, 0

	for p.pos < len(p.content) {
		if p.pos >= len(p.content) {
			break
		}
		if preformattedRangeStart >= 0 && !p.preformattedRangeStartContinues() {
			preformattedRangeStart = -1
		}
		if suppressInitialPreformattedLF {
			suppressed, _, start, line, column := p.consumeInitialPreformattedLF(&suppressInitialPreformattedLF)
			if suppressed {
				// Keep this tentative boundary across tokenizer input that emits no
				// token (notably exact </>). The next loop either applies it to a
				// leading LF character token or retires it on any emitted token.
				preformattedRangeStart = start
				preformattedRangeLine, preformattedRangeColumn = line, column
				continue
			}
		}
		if consumed, err := p.consumePendingAdoptionEnd(node); consumed || err != nil {
			if err != nil {
				return nil, err
			}
			continue
		}
		if p.currentTemplateEndCloses(node) {
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		if p.currentTemplateTableTransitionWillEmit(node) {
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		if name, closing, ok := p.peekTagName(); ok && closing {
			if owner, blocked := p.matchingForeignAncestorWithinIntegration(node, name); owner != nil {
				if blocked {
					start := p.pos
					if _, emitted, err := p.parseClosingTag(); err != nil {
						return nil, err
					} else if !emitted {
						p.extendPreviousTextRange(node, start)
					}
					continue
				}
				p.finishImplicitElement(node, textContent.String(), contentStartPos)
				return node, nil
			}
			if name != node.Name && p.foreignIgnoredEnds[name] > 0 && nearestOpenElementByName(node, name) == nil {
				start := p.pos
				if _, emitted, err := p.parseClosingTag(); err != nil {
					return nil, err
				} else if !emitted {
					p.extendPreviousTextRange(node, start)
					p.recoverOpenElementsAtEOF = true
				}
				p.retireForeignIgnoredEnd(name)
				continue
			}
			if name != node.Name && p.foreignIgnoredHTMLEnds[name] > 0 && nearestOpenElementByName(node, name) == nil {
				start := p.pos
				if _, emitted, err := p.parseClosingTag(); err != nil {
					return nil, err
				} else if !emitted {
					p.extendPreviousTextRange(node, start)
				}
				p.foreignIgnoredHTMLEnds[name]--
				continue
			}
			if p.foreignBreakoutRecovery && name != node.Name && nearestOpenElementByName(node.Parent, name) != nil && !isHTMLSpecialElement(node.Name) {
				p.foreignIgnoredHTMLEnds[node.Name]++
				p.finishImplicitElement(node, textContent.String(), contentStartPos)
				return node, nil
			}
			if p.foreignIgnoredEndCount > 0 && name != node.Name && foreignRecoveryGenericEndTarget(node, name) != nil {
				p.foreignIgnoredHTMLEnds[node.Name]++
				p.finishImplicitElement(node, textContent.String(), contentStartPos)
				return node, nil
			}
		}
		if p.explicitDocumentSkeleton {
			if name, closing, ok := p.peekTagName(); ok && !closing && p.currentTagTokenWillEmit() {
				switch name {
				case "html":
					if err := p.mergeDuplicateStartInto(p.documentHTML, "html"); err != nil {
						return nil, err
					}
					continue
				case "body":
					if p.documentBody != nil && p.activeSelect == nil {
						if err := p.mergeDuplicateStartInto(p.documentBody, "body"); err != nil {
							return nil, err
						}
						continue
					}
				case "head":
					if p.documentHead != nil {
						p.consumeIgnoredStartTag()
						continue
					}
				}
			}
		}
		if p.templateOwnerForNode(node) != nil {
			if name, closing, tag := p.peekTagName(); tag && !closing && (name == "frame" || name == "frameset") && p.currentTagTokenWillEmit() {
				p.consumeIgnoredStartTag()
				continue
			}
		}
		if handled, err := p.handleForeignDocumentStart(); handled || err != nil {
			if err != nil {
				return nil, err
			}
			continue
		}
		if name, closing, tag := p.peekTagName(); tag && closing && name == "form" && node != p.activeForm && isImpliedEndTagName(node.Name) && p.activeForm != nil && formattingInOrdinaryScope(p.activeForm, node) {
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		if handled, err := p.handleFormToken(node, p.tableFosterMode > 0); handled || err != nil {
			if err != nil {
				return nil, err
			}
			if p.closedForms[node] {
				delete(p.formEOFBoundaries, node)
				refreshNodeTextContent(node)
				return node, nil
			}
			continue
		}
		if (p.tableFosterMode > 0 && p.currentFosterTransitionWillEmit()) || (p.tableCellMode > 0 && p.currentCellTransitionWillEmit()) {
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		if p.tableFosterMode > 0 && p.currentFosterIgnoredEndTag() {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if p.tableCellMode > 0 && p.currentCellIgnoredEndTag() {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if p.tableCaptionMode > 0 && p.currentCaptionIgnoredEndTag() {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if p.currentNestedButtonStart(node) {
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		if p.tableCaptionMode > 0 && p.currentCaptionTransitionWillEmit() {
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		// The in-table rules insert an input whose parsed type is exactly
		// ASCII-case-insensitive "hidden" at the current node and immediately
		// pop it. This must happen before normal in-body reconstruction/foster
		// parenting, including while an ordinary descendant is being parsed.
		if p.tableFosterMode > 0 {
			hiddenInput, consumed, err := p.parseHiddenInputInTable(node)
			if err != nil {
				return nil, err
			}
			if consumed {
				if hiddenInput != nil {
					appendUniqueChild(node, hiddenInput)
				}
				continue
			}
		}
		if p.listDocumentEndMode != "" && (p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction()) {
			commentParent := p.documentHTML
			if p.listDocumentEndMode == "html" {
				commentParent = p.documentRoot
			}
			comment, err := p.parseNode(commentParent)
			if err != nil {
				return nil, err
			}
			if comment != nil {
				comment.Parent = commentParent
				if p.listDocumentEndMode == "html" {
					p.deferredRootComments = append(p.deferredRootComments, comment)
				} else {
					p.deferredHTMLComments = append(p.deferredHTMLComments, comment)
				}
			}
			continue
		}
		if p.listDocumentEndMode != "" && p.nextTokenIsIgnoredDoctype() {
			start, line, column := p.pos, p.line, p.col
			if _, err := p.parseDoctype(node, start, line, column); err != nil {
				return nil, err
			}
			if p.pos == start {
				return nil, fmt.Errorf("failed to consume doctype at position %d", start)
			}
			continue
		}
		if p.listDocumentEndMode != "" && p.nextTokenIsHTMLStart() {
			if err := p.mergeDuplicateHTMLStart(); err != nil {
				return nil, err
			}
			continue
		}
		if p.currentListDocumentEndTag() {
			needsOpenRecovery := p.hasActiveListItem() || len(p.activeFormatting) > 0
			tokenStart := p.pos
			tokenLine, tokenColumn := p.line, p.col
			name, _, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if name == "body" && p.documentBody != nil {
				p.listDocumentEndMode = "body"
				p.documentBodyContentEnd = tokenStart
				p.documentBodyEndPos = p.pos
				p.documentBodyEndLine = p.line
				p.documentBodyEndColumn = p.col
			}
			if name == "html" && p.documentHTML != nil {
				p.listDocumentEndMode = "html"
				p.documentHTMLContentEnd = tokenStart
				p.documentHTMLEndPos = p.pos
				p.documentHTMLEndLine = p.line
				p.documentHTMLEndColumn = p.col
				if p.documentBody != nil && p.documentBodyEndPos == 0 {
					p.documentBodyContentEnd = tokenStart
					p.documentBodyEndPos = tokenStart
					p.documentBodyEndLine = tokenLine
					p.documentBodyEndColumn = tokenColumn
				}
			}
			if len(p.activeFormatting) > 0 && !p.formattingDocumentEnd {
				p.formattingDocumentEnd = true
				p.formattingDocumentEndPrior = p.recoverOpenElementsAtEOF
			}
			if needsOpenRecovery {
				p.recoverOpenElementsAtEOF = true
			}
			continue
		}
		if p.listDocumentEndMode != "" && p.currentTokenReentersListBody() {
			p.listDocumentEndMode = ""
		}
		if p.activeSelect != nil {
			name, closing, tag := p.peekTagName()
			// The select insertion mode ignores noscript tags in both scripting
			// configurations; their contents are still processed as select tokens.
			if tag && name == "noscript" && p.currentTagTokenWillEmit() {
				if closing {
					if _, _, err := p.parseClosingTag(); err != nil {
						return nil, err
					}
				} else {
					p.consumeIgnoredStartTag()
				}
				continue
			}
			if tag && !closing && (name == "html" || name == "body" || name == "head") && p.currentTagTokenWillEmit() {
				tokenStart, tokenLine, tokenColumn := p.pos, p.line, p.col
				var err error
				switch name {
				case "html":
					err = p.mergeDuplicateStartInto(p.documentHTML, "html")
				case "body":
					p.consumeIgnoredStartTag()
				case "head":
					p.consumeIgnoredStartTag()
				}
				if err != nil {
					return nil, err
				}
				p.preserveIgnoredTextRange(node, tokenStart, tokenLine, tokenColumn)
				continue
			}
			// Bare table-structure starts are parse errors in a customizable
			// select and their wrappers are ignored. Specialized table/cell and
			// template table-derived transitions have already unwound above; in
			// ordinary in-body select parsing, retain only the following content.
			if tag && !closing && isTableStructureStart(name) && p.currentTagTokenWillEmit() {
				p.consumeIgnoredStartTag()
				continue
			}
		}
		// Active select identities are dynamically scoped to the recursive open
		// path, so a cached target is necessarily an ancestor of this frame.
		// Avoid re-walking that path in every descendant during one unwind.
		if target, action := p.selectTreeUnwindTarget(node); target != nil {
			if node == target {
				tokenStart := p.pos
				tokenLine, tokenColumn := p.line, p.col
				switch action {
				case "end":
					_, emitted, err := p.parseClosingTag()
					if err != nil {
						return nil, err
					}
					if !emitted {
						p.extendPreviousTextRange(node, tokenStart)
						p.recoverOpenElementsAtEOF = true
						p.finishImplicitElement(node, textContent.String(), contentStartPos)
						p.clearSelectUnwindCache()
						return node, nil
					}
					node.TextContent = textContent.String()
					node.ContentStart = contentStartPos
					node.ContentEnd = tokenStart
					node.EndPos = p.pos
					node.EndLine = p.line
					node.EndColumn = p.col
					p.clearSelectUnwindCache()
					return node, nil
				case "nested-select":
					if !p.consumeIgnoredStartTag() {
						p.finishImplicitElement(node, textContent.String(), contentStartPos)
						p.clearSelectUnwindCache()
						return node, nil
					}
					p.finishElementAt(node, textContent.String(), contentStartPos, tokenStart, tokenLine, tokenColumn)
					p.clearSelectUnwindCache()
					return node, nil
				}
				p.clearSelectUnwindCache()
			}
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		if p.currentIgnoredSelectEndTag(node) {
			tokenStart := p.pos
			tokenLine, tokenColumn := p.line, p.col
			ignoredName, _, _ := p.peekTagName()
			if _, emitted, err := p.parseClosingTag(); err != nil {
				return nil, err
			} else if !emitted {
				p.extendPreviousTextRange(node, tokenStart)
			} else {
				p.preserveIgnoredTextRange(node, tokenStart, tokenLine, tokenColumn)
				var outside *types.Node
				if p.activeSelect != nil {
					outside = nearestOpenElementByName(p.activeSelect.Parent, ignoredName)
				}
				if outside != nil {
					p.recoverOpenElementsAtEOF = true
					p.selectDocumentEndRecovery = p.activeSelect
				}
			}
			continue
		}
		if target, explicit := p.listItemUnwindTarget(node); target != nil {
			if currentIsTarget := node == target; currentIsTarget && explicit {
				tokenStart := p.pos
				name, emitted, err := p.parseClosingTag()
				if err != nil {
					return nil, err
				}
				if !emitted || name != target.Name {
					p.extendPreviousTextRange(node, tokenStart)
					p.recoverOpenElementsAtEOF = true
					p.finishImplicitElement(node, textContent.String(), contentStartPos)
					return node, nil
				}
				node.TextContent = textContent.String()
				node.ContentStart = contentStartPos
				node.ContentEnd = tokenStart
				node.EndPos = p.pos
				node.EndLine = p.line
				node.EndColumn = p.col
				if target.Parent != nil && (target.Parent.Name == "ul" || target.Parent.Name == "ol" || target.Parent.Name == "dl") {
					p.recoverOpenElementsAtEOF = true
					p.listEOFRecoveryBoundary = target.Parent
				}
				return node, nil
			}
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			if node == target {
				p.clearListUnwindCache()
			}
			return node, nil
		}
		if p.currentIgnoredListEndTag(node) {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if name, closing, ok := p.peekTagName(); ok && closing && p.adoptionIgnoredEnds[name] > 0 && p.activeFormattingEntryByName(name) == nil && nearestOpenElementByName(node, name) == nil {
			start := p.pos
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			p.extendPreviousTextRange(node, start)
			p.adoptionIgnoredEnds[name]--
			continue
		}
		if node.Name == "html" && p.documentEndRecovery && p.nextTokenIsProcessingInstruction() {
			instruction, err := p.parseNode(p.documentRoot)
			if err != nil {
				return nil, err
			}
			if instruction != nil {
				instruction.Parent = p.documentRoot
				p.deferredRootComments = append(p.deferredRootComments, instruction)
			}
			continue
		}
		if node.Name == "html" && p.documentEndRecovery && p.pendingDocumentEndTarget() == nil && p.documentBody != nil {
			name, closing, tag := p.peekTagName()
			if !tag || !closing || (name != "html" && name != "body") {
				insertionParent := p.formattingInsertionParent(p.documentBody, p.nextTokenReconstructsFormatting())
				child, err := p.parseNode(insertionParent)
				if err != nil {
					return nil, err
				}
				if child != nil {
					if insertionParent != p.documentBody {
						p.appendFormattingChild(p.documentBody, insertionParent, child)
						refreshNodeTextContent(p.documentBody)
						refreshNodeTextContent(p.documentHTML)
					} else {
						p.appendToDocumentBody(child)
					}
					textContent.WriteString(child.TextContent)
				}
				continue
			}
		}
		if node.Name == "html" && p.pendingDocumentEndTarget() != nil {
			if p.pendingDocumentEndBlockStarts() {
				parent := p.resolvePendingThroughParagraph()
				if parent == nil {
					parent = p.documentBody
				}
				child, err := p.parseNode(parent)
				if err != nil {
					return nil, err
				}
				if child != nil {
					if p.pendingDocumentEndTarget() != nil {
						p.appendToPendingDocumentEnd(p.pendingDocumentEndTarget(), child)
					} else {
						p.appendToDocumentBody(child)
					}
					textContent.WriteString(child.TextContent)
				}
				continue
			}
			if name, closing, ok := p.peekTagName(); ok && !closing && name == "table" && p.quirksMode && p.currentTagTokenWillEmit() {
				var paragraph *types.Node
				for _, pending := range p.nodesClosedAtDocumentEnd {
					if pending.Name == "p" {
						paragraph = pending
						break
					}
				}
				if paragraph != nil {
					child, err := p.parseNode(paragraph)
					if err != nil {
						return nil, err
					}
					if child != nil {
						p.appendToPendingDocumentEnd(paragraph, child)
						textContent.WriteString(child.TextContent)
					}
					continue
				}
			}
			if name, closing, ok := p.peekTagName(); ok && closing {
				if resolved, err := p.resolvePendingThroughName(name, true); err != nil {
					return nil, err
				} else if resolved {
					continue
				}
			}
			if p.peek() != '<' || !p.startsMarkupToken() {
				child, err := p.parseNode(p.pendingDocumentEndTarget())
				if err != nil {
					return nil, err
				}
				if child != nil {
					p.appendToPendingDocumentEnd(p.pendingDocumentEndTarget(), child)
					textContent.WriteString(child.TextContent)
				}
				continue
			}
		}
		if p.paragraphAncestorClose != "" {
			name, closing, ok := p.peekTagName()
			if ok && closing && name == p.paragraphAncestorClose {
				if node.Name != name {
					if (name == "html" || name == "body") && node.Name != "html" && node.Name != "body" {
						p.nodesClosedAtDocumentEnd = append(p.nodesClosedAtDocumentEnd, node)
						p.documentEndRecovery = true
					}
					p.finishImplicitElement(node, textContent.String(), contentStartPos)
					return node, nil
				}
				p.paragraphAncestorClose = ""
			}
		}

		// HTML's tree-construction rules implicitly close the current table row
		// when another row/section begins or the containing table/section ends.
		// Leave the triggering tag unconsumed so the parent can process it.
		if p.shouldImplicitlyCloseTableRow(node.Name) {
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}

		// In the "in body" insertion mode, block-like start tags close a p
		// in button scope before the triggering token is processed. Descendant
		// inline frames unwind the same way, leaving the token for the p's
		// parent. An ancestor end tag and an explicit p end tag do likewise.
		if p.shouldUnwindForParagraph(node) {
			name, closing, _ := p.peekTagName()
			if closing && (name == "html" || name == "body") && node.Name != "html" && node.Name != "body" {
				p.nodesClosedAtDocumentEnd = append(p.nodesClosedAtDocumentEnd, node)
				p.documentEndRecovery = true
			}
			if node == p.activeParagraph {
				if closing && name != "p" {
					p.paragraphAncestorClose = name
				}
			}
			p.finishImplicitElement(node, textContent.String(), contentStartPos)
			return node, nil
		}
		// Body/html end tags change the document insertion mode without
		// discarding ordinary descendants that are still logically open. Keep
		// those frames in the pending document-end stack so after-body content
		// is reprocessed through them. Specialized paragraph/list/select paths
		// above get first refusal because they also maintain their own identity
		// and source-range state.
		if p.explicitDocumentSkeleton {
			if name, closing, ok := p.peekTagName(); ok && closing && (name == "body" || name == "html") {
				insideBody := p.documentBody != nil && node != p.documentBody && nodeWithin(node, p.documentBody)
				insideHTML := p.documentHTML != nil && node != p.documentHTML && nodeWithin(node, p.documentHTML)
				if (name == "body" && insideBody) || (name == "html" && insideHTML) {
					if name != "html" || node != p.documentBody {
						seen := false
						for _, pending := range p.nodesClosedAtDocumentEnd {
							if pending == node {
								seen = true
								break
							}
						}
						if !seen {
							p.nodesClosedAtDocumentEnd = append(p.nodesClosedAtDocumentEnd, node)
						}
						p.documentEndRecovery = true
					}
					p.finishImplicitElement(node, textContent.String(), contentStartPos)
					return node, nil
				}
			}
		}
		if action, err := p.handleSpecialFormattingStart(node, textContent.String(), contentStartPos); err != nil {
			return nil, err
		} else {
			switch action {
			case adoptionContinue:
				continue
			case adoptionReturn:
				return node, nil
			}
		}
		if name, closing, ok := p.peekTagName(); ok && closing && isCoreAdoptionSubject(name) {
			action, err := p.handleCoreAdoption(node, name, textContent.String(), contentStartPos, true)
			if err != nil {
				return nil, err
			}
			switch action {
			case adoptionContinue:
				continue
			case adoptionReturn:
				return node, nil
			}
		}
		if name, closing, ok := p.peekTagName(); ok && closing && isFormattingElement(name) {
			if entry := p.activeFormattingEntryByName(name); entry != nil && entry.current != nil && entry.current != node && nodeWithin(entry.current, node) {
				tokenStart := p.pos
				if _, emitted, err := p.parseClosingTag(); err != nil {
					return nil, err
				} else if !emitted {
					p.extendPreviousTextRange(entry.current, tokenStart)
				}
				p.refreshFormattingChain(entry.current)
				entry.current.ContentEnd = tokenStart
				entry.current.EndPos = p.pos
				entry.current.EndLine = p.line
				entry.current.EndColumn = p.col
				p.removeActiveFormatting(entry.current)
				p.recoverOpenElementsAtEOF = true
				continue
			}
		}
		if isFormattingElement(node.Name) {
			name, closing, ok := p.peekTagName()
			if ok && closing && name == node.Name {
				if active := p.formattingEntryForNode(node); active != nil && active.current != node {
					tokenStart := p.pos
					if _, emitted, err := p.parseClosingTag(); err != nil {
						return nil, err
					} else if !emitted {
						p.extendPreviousTextRange(node, tokenStart)
					}
					p.removeActiveFormatting(node)
					p.finishImplicitElement(node, textContent.String(), contentStartPos)
					return node, nil
				}
			}
			if ok && closing && name != node.Name && isParagraphClosingAncestorEndTag(name) && nearestOpenElementByName(node.Parent, name) != nil {
				p.finishImplicitElement(node, textContent.String(), contentStartPos)
				return node, nil
			}
		}

		// Complete in-body generic/special end-tag recovery before falling into
		// the recursive parser's strict matched-end path. Special block/list
		// ends pop every intervening node when their target is in ordinary scope;
		// any-other ends pop only through non-special nodes. If no eligible target
		// exists, the emitted token is ignored. Leaving a target token unconsumed
		// lets each recursive frame unwind naturally and the matching ancestor
		// consume it with the correct source range.
		if name, closing, ok := p.peekTagName(); ok && closing && name != node.Name && name != "p" && name != "br" && name != "script" &&
			!p.isBogusEndTagOpen() && !p.isMissingEndTagName() && p.foreignHTMLDepth == 0 &&
			(p.activeSelect == nil || !nodeIsDescendantOrSelf(node, p.activeSelect)) &&
			(!isHeadingName(node.Name) || !isHeadingName(name)) && p.currentTagTokenWillEmit() {
			if p.activeParagraph != nil {
				if _, isAncestor := p.activeParagraphAncestors[name]; isAncestor && !isParagraphClosingAncestorEndTag(name) {
					// The established paragraph recovery path below records scoped
					// EOF state for this ignored ancestor end.
					goto strictMatchedEndPath
				}
			}
			target := p.cachedGenericEndTarget(node, name)
			if target != nil {
				p.finishImplicitElement(node, textContent.String(), contentStartPos)
				return node, nil
			}
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}

	strictMatchedEndPath:
		// Check for closing tag
		if p.peek() == '<' && p.pos+1 < len(p.content) && p.content[p.pos+1] == '/' && p.startsMarkupToken() {
			if p.isBogusEndTagOpen() {
				commentStart := p.pos
				commentLine := p.line
				commentColumn := p.col
				comment, err := p.parseBogusEndTagComment(node, commentStart, commentLine, commentColumn)
				if err != nil {
					return nil, err
				}
				node.Children = append(node.Children, comment)
				continue
			}
			if p.isMissingEndTagName() {
				p.advanceRune(3) // Ignore "</>" and merge surrounding text.
				continue
			}
			if recovered, consumed, err := p.consumeVoidElementClosingTag(node); consumed || err != nil {
				if err != nil {
					return nil, err
				}
				if recovered != nil {
					node.Children = append(node.Children, recovered)
					textContent.WriteString(recovered.TextContent)
				}
				continue
			}

			// Mark content end position before closing tag
			contentEndPos := p.pos
			contentEndLine, contentEndColumn := p.line, p.col
			closingTag, emitted, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if !emitted {
				p.extendPreviousTextRange(node, contentEndPos)
				node.TextContent = textContent.String()
				node.ContentStart = contentStartPos
				node.ContentEnd = contentEndPos
				node.EndPos = p.pos
				node.EndLine = p.line
				node.EndColumn = p.col
				return node, nil
			}
			if closingTag == node.Name {
				p.retireForeignIgnoredEnd(closingTag)
				if !isFormattingElement(node.Name) {
					p.finishFormattingWithin(node, contentEndPos, contentEndLine, contentEndColumn)
				}
				node.TextContent = textContent.String()
				node.ContentStart = contentStartPos
				node.ContentEnd = contentEndPos
				node.EndPos = p.pos
				node.EndLine = p.line
				node.EndColumn = p.col
				if p.listEOFRecoveryBoundary == node {
					p.listEOFRecoveryBoundary = nil
					p.recoverOpenElementsAtEOF = false
				}
				if p.selectDocumentEndRecovery != nil && nodeIsDescendantOrSelf(p.selectDocumentEndRecovery, node) {
					p.selectDocumentEndRecovery = nil
					p.recoverOpenElementsAtEOF = false
				}
				if isFormattingElement(node.Name) {
					p.removeActiveFormatting(node)
				}
				if p.adoptionEOFBoundary == node {
					p.adoptionEOFBoundary = nil
					p.recoverOpenElementsAtEOF = false
				}
				return node, nil
			}
			if isHeadingName(node.Name) && isHeadingName(closingTag) {
				p.finishElementAt(node, textContent.String(), contentStartPos, contentEndPos, contentEndLine, contentEndColumn)
				return node, nil
			}
			// An end tag for script is ignored when no script element is in
			// scope, including while parsing an explicit body or another
			// ordinary ancestor.
			if closingTag == "script" {
				continue
			}
			if closingTag == "p" && p.paragraphInButtonScope(node) == nil {
				if p.headDepth > 0 && !p.headExited {
					continue
				}
				if node.Name == "html" && !p.bodyContentStarted {
					continue
				}
				node.Children = append(node.Children, syntheticParagraph(node))
				continue
			}
			// A table is an ordinary-scope boundary for customizable selects.
			// An end-select token inside that table cannot reach an outer select
			// and is ignored by the in-body generic end-tag algorithm.
			if closingTag == "select" && p.selectBlockedByTable {
				continue
			}
			if p.listItemBlocksGenericAncestorEnd(node, closingTag) {
				continue
			}
			if p.listItemIgnoresAbsentAncestorEnd(node, closingTag) {
				continue
			}
			// A customizable select is an ordinary-scope boundary. End tags for
			// elements outside it are ignored while the select remains open.
			if p.activeSelect != nil && nodeIsDescendantOrSelf(node, p.activeSelect) {
				outside := nearestOpenElementByName(p.activeSelect.Parent, closingTag)
				if outside != nil {
					p.preserveIgnoredTextRange(node, contentEndPos, contentEndLine, contentEndColumn)
					p.recoverOpenElementsAtEOF = true
					p.selectDocumentEndRecovery = p.activeSelect
					continue
				}
			}
			if p.activeParagraph != nil {
				if _, isAncestor := p.activeParagraphAncestors[closingTag]; isAncestor && !isParagraphClosingAncestorEndTag(closingTag) {
					p.recoverOpenElementsAtEOF = true
					continue
				}
			}
			if p.foreignHTMLDepth > 0 {
				// HTML descendants of an SVG integration point use the ordinary
				// in-body generic end-tag algorithm, but the integration owner is a
				// special barrier. An absent or outside matching name is ignored.
				continue
			}
			return nil, fmt.Errorf("expected closing tag </%s>, got </%s> at position %d", node.Name, closingTag, contentEndPos)
		}

		formattingContainer := node
		if pending := p.adoptionPendingEnds[node]; len(pending) > 0 {
			formattingContainer = pending[0]
		}
		if p.documentEndRecovery && node == p.documentHTML && p.documentBody != nil {
			formattingContainer = p.documentBody
		}
		reconstructFormatting := p.nextTokenReconstructsFormatting()
		insertionParent := formattingContainer
		if !p.nextTokenIsProcessingInstruction() {
			insertionParent = p.formattingInsertionParent(formattingContainer, reconstructFormatting)
		}
		child, err := p.parseNode(insertionParent)
		if err != nil {
			return nil, err
		}
		if preformattedRangeStart >= 0 && child != nil {
			if child.Type == types.TextNode && strings.HasPrefix(child.Value, "\n") {
				child.StartPos = preformattedRangeStart
				child.StartLine = preformattedRangeLine
				child.StartColumn = preformattedRangeColumn
			}
			preformattedRangeStart = -1
		}
		if child != nil {
			_, splitCurrent := p.adoptionSplit[node]
			detachedChild := p.adoptionDetached[child]
			if insertionParent != node {
				p.appendFormattingChild(node, insertionParent, child)
				if child.Type != types.CommentNode && child.Type != types.ProcessingInstructionNode {
					textContent.WriteString(child.TextContent)
				}
			}
			if child.Type == types.TextNode && !isHTMLWhitespaceString(child.Value) {
				p.bodyContentStarted = true
				if p.headDepth > 0 {
					p.headExited = true
				}
			}
			if detachedChild {
				delete(p.adoptionDetached, child)
			} else if insertionParent != node {
				// The child is already attached through reconstructed formatting
				// wrappers; only the ancestor's aggregate text remains to update.
			} else if child.Type == types.TextNode && len(node.Children) > 0 && node.Children[len(node.Children)-1].Type == types.TextNode {
				mergeTextNodes(node.Children[len(node.Children)-1], child)
			} else {
				node.Children = append(node.Children, child)
			}
			fosteredText := ""
			if !detachedChild {
				fosteredText = p.appendDeferredTableFoster(node, child)
				p.appendDeferredListComments(node, child)
				p.appendDeferredAdoption(node, child)
				p.appendDeferredFormSiblings(node, child)
			}
			if insertionParent == node && !detachedChild {
				switch child.Type {
				case types.TextNode:
					textContent.WriteString(child.Value)
				case types.ElementNode:
					// Fostered table content precedes the table in the DOM even
					// though its source token occurred after the table start tag.
					textContent.WriteString(fosteredText)
					textContent.WriteString(child.TextContent)
				}
			}
			if splitCurrent {
				node.TextContent = textContent.String()
				return node, nil
			}
			if p.adoptionClosed[node] {
				delete(p.adoptionClosed, node)
				refreshNodeTextContent(node)
				return node, nil
			}
			if p.closedForms[node] {
				delete(p.formEOFBoundaries, node)
				refreshNodeTextContent(node)
				return node, nil
			}
		}
		if p.adoptionReturnAtSplit[node] {
			delete(p.adoptionReturnAtSplit, node)
			delete(p.formattingText, node)
			refreshNodeTextContent(node)
			return node, nil
		}
	}
	p.finishPendingAdoptionAtEOF(node)
	documentBoundaryClosed := (node == p.documentBody && p.documentBodyEndPos > 0) ||
		(node == p.documentHTML && (p.documentHTMLEndPos > 0 || p.documentBodyEndPos > 0))
	formRecovery := false
	for boundary := range p.formEOFBoundaries {
		if nodeIsDescendantOrSelf(node, boundary) {
			formRecovery = true
			break
		}
	}
	foreignRecovery := p.foreignHTMLDepth > 0 || (p.foreignBreakoutBoundary != nil && nodeIsDescendantOrSelf(node, p.foreignBreakoutBoundary))
	// parseElement is only entered through the document-mode driver once a
	// source-backed or synthetic HTML element exists. Therefore every open
	// ordinary frame belongs to this document, and EOF recovery is O(1). A
	// recursive nodeWithin check here makes a depth-N EOF path quadratic.
	explicitDocumentEOF := p.explicitDocumentSkeleton && p.documentHTML != nil
	if p.recoverOpenElementsAtEOF || explicitDocumentEOF || foreignRecovery || documentBoundaryClosed || formRecovery || p.openForms[node] || p.paragraphInButtonScope(node) != nil || p.hasActiveListItem() || p.hasActiveSelectTree() || len(p.activeFormatting) > 0 {
		if len(p.activeFormatting) > 0 {
			p.recoverOpenElementsAtEOF = true
		}
		if p.hasActiveSelectTree() {
			p.recoverOpenElementsAtEOF = true
		}
		if p.hasActiveListItem() {
			p.recoverOpenElementsAtEOF = true
		}
		p.finishImplicitElement(node, textContent.String(), contentStartPos)
		return node, nil
	}

	return nil, fmt.Errorf("unterminated element <%s> at position %d", node.Name, startPos)
}

func (p *HTMLParser) hasActiveSelectTree() bool {
	return p.activeSelect != nil || p.activeOptionEnd != nil || p.activeOptgroupEnd != nil
}

func (p *HTMLParser) currentTemplate() *types.Node {
	if len(p.activeTemplates) == 0 {
		return nil
	}
	return p.activeTemplates[len(p.activeTemplates)-1]
}

// currentTemplateEndCloses reports whether the next token closes the template
// whose inert fragment owns the current recursive frame. The token remains
// unconsumed so frames can unwind to the template driver one at a time.
func (p *HTMLParser) currentTemplateEndCloses(node *types.Node) bool {
	owner := p.currentTemplate()
	if owner == nil || p.templateOwnerForNode(node) != owner {
		return false
	}
	name, closing, ok := p.peekTagName()
	return ok && closing && name == "template" && p.currentTagTokenWillEmit()
}

// currentTemplateTableTransitionWillEmit reports a structural start token
// that must be reprocessed by the persistent table-derived insertion mode of
// the innermost template. Generic descendant frames must unwind before the
// template driver can apply that mode; otherwise elements such as div/select
// incorrectly swallow rows and cells.
func (p *HTMLParser) currentTemplateTableTransitionWillEmit(node *types.Node) bool {
	if len(p.templateInsertionModes) == 0 || p.templateOwnerForNode(node) != p.currentTemplate() {
		return false
	}
	mode := p.templateInsertionModes[len(p.templateInsertionModes)-1]
	if mode != templateModeTable && mode != templateModeTableBody && mode != templateModeRow {
		return false
	}
	name, closing, ok := p.peekTagName()
	return ok && !closing && isTableStructuralName(name) && p.currentTagTokenWillEmit()
}

func (p *HTMLParser) templateOwnerForNode(node *types.Node) *types.Node {
	if node == nil {
		return nil
	}
	if p.templateOwnerKnown[node] {
		return p.templateOwner[node]
	}
	path := make([]*types.Node, 0, 4)
	var owner *types.Node
	for ancestor := node; ancestor != nil; ancestor = ancestor.Parent {
		if p.templateOwnerKnown[ancestor] {
			owner = p.templateOwner[ancestor]
			break
		}
		if owner := p.templateOwner[ancestor]; owner != nil {
			for _, visited := range path {
				p.templateOwner[visited] = owner
				p.templateOwnerKnown[visited] = true
			}
			p.templateOwnerKnown[ancestor] = true
			return owner
		}
		path = append(path, ancestor)
	}
	for _, visited := range path {
		if owner != nil {
			p.templateOwner[visited] = owner
		}
		p.templateOwnerKnown[visited] = true
	}
	return owner
}

func newTemplateContent(host *types.Node) *types.Node {
	return &types.Node{
		Type:     types.DocumentFragmentNode,
		Name:     "#document-fragment",
		Children: []*types.Node{},
	}
}

func (p *HTMLParser) appendTemplateChild(fragment, child *types.Node) {
	if child == nil {
		return
	}
	child.Parent = fragment
	if child.Type == types.TextNode && len(fragment.Children) > 0 && fragment.Children[len(fragment.Children)-1].Type == types.TextNode {
		mergeTextNodes(fragment.Children[len(fragment.Children)-1], child)
		return
	}
	fragment.Children = append(fragment.Children, child)
}

type templateInsertionMode uint8

const (
	templateModeInitial templateInsertionMode = iota
	templateModeBody
	templateModeTable
	templateModeColumnGroup
	templateModeTableBody
	templateModeRow
)

// parseTemplateElement parses into an inert DocumentFragment. The template is
// a leaf in the document tree; recursive parser scope is tracked separately in
// templateOwner because DOM Parent intentionally stops at the fragment.
func (p *HTMLParser) parseTemplateElement(template *types.Node, contentStart int) (*types.Node, error) {
	fragment := newTemplateContent(template)
	template.TemplateContent = fragment
	p.templateOwner[fragment] = template
	p.templateOwnerKnown[fragment] = true
	p.activeTemplates = append(p.activeTemplates, template)
	p.templateInsertionModes = append(p.templateInsertionModes, templateModeInitial)
	defer func() {
		delete(p.templateOwner, fragment)
		p.activeTemplates = p.activeTemplates[:len(p.activeTemplates)-1]
		p.templateInsertionModes = p.templateInsertionModes[:len(p.templateInsertionModes)-1]
	}()

	// The form element pointer is scoped outside template contents. Forms in the
	// inert subtree may use ordinary pointer recovery without replacing or
	// clearing the pointer belonging to the surrounding document tree.
	outerForm := p.activeForm
	p.activeForm = nil
	defer func() { p.activeForm = outerForm }()
	mode := templateModeInitial
	setMode := func(next templateInsertionMode) {
		mode = next
		p.templateInsertionModes[len(p.templateInsertionModes)-1] = next
	}
	lastBoundary := template.StartPos
	lastBoundaryLine, lastBoundaryColumn := template.StartLine, template.StartColumn
	updateIgnoredStartBoundary := func() {
		lastBoundary = p.pos
		lastBoundaryLine, lastBoundaryColumn = p.line, p.col
	}

	for p.pos < len(p.content) {
		if p.templateForeignTransition {
			p.templateForeignTransition = false
		}
		name, closing, tag := p.peekTagName()
		if tag && !p.currentTagTokenWillEmit() {
			child, err := p.parseNode(fragment)
			if err != nil {
				return nil, err
			}
			p.appendTemplateChild(fragment, child)
			continue
		}
		if tag && closing && name == "template" {
			contentEnd := p.pos
			if _, emitted, err := p.parseClosingTag(); err != nil {
				return nil, err
			} else if !emitted {
				p.extendPreviousTextRange(fragment, contentEnd)
				break
			}
			template.ContentStart = contentStart
			template.ContentEnd = contentEnd
			template.EndPos = p.pos
			template.EndLine = p.line
			template.EndColumn = p.col
			p.refreshOpenFormattingText()
			refreshNodeTextContent(fragment)
			return template, nil
		}
		if tag && closing && name == "table" {
			// A table outside the template is not in the template's table scope.
			// The in-template rule ignores this end tag.
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if mode == templateModeInitial && tag && !closing && (name == "html" || name == "head" || name == "body") && p.currentTagTokenWillEmit() {
			setMode(templateModeBody)
			continue
		}
		if tag && !closing && p.currentTagTokenWillEmit() {
			switch name {
			case "html":
				updateIgnoredStartBoundary()
				p.consumeIgnoredStartTag()
				continue
			case "head", "body":
				updateIgnoredStartBoundary()
				p.consumeIgnoredStartTag()
				continue
			}
		}
		if tag && closing && name != "p" && name != "br" {
			// With every content descendant already unwound, an end tag seen by
			// the template driver has no matching element in template scope.
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if tag && !closing && (name == "frame" || name == "frameset") && p.currentTagTokenWillEmit() {
			updateIgnoredStartBoundary()
			p.consumeIgnoredStartTag()
			continue
		}

		// The stack of template insertion modes is persistent. Once a token
		// switches away from "in template", later tokens are processed using that
		// mode until a table transition changes it again; dispatching every root
		// token independently produces the wrong wrappers and ignored-token rules.
		if mode == templateModeInitial {
			if !tag || p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() || p.nextTokenIsIgnoredDoctype() || (tag && !closing && isHeadOnlyElement(name) && name != "head") {
				// Character/comment/doctype and in-head tokens do not replace the
				// current template insertion mode.
			} else if closing {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				continue
			} else {
				switch name {
				case "caption", "colgroup", "tbody", "tfoot", "thead":
					setMode(templateModeTable)
				case "col":
					setMode(templateModeColumnGroup)
				case "tr":
					setMode(templateModeTableBody)
				case "td", "th":
					setMode(templateModeRow)
				default:
					setMode(templateModeBody)
				}
				continue
			}
		}

		// A form start is processed using the in-body rule only after the
		// template insertion-mode stack has switched to in body. In table and
		// table-derived template modes, the in-table rule ignores a form start
		// while a template is open. Form end tags still use the ordinary pointer
		// recovery rule in every mode.
		if name == "form" && (closing || mode == templateModeBody) {
			if handled, err := p.handleFormToken(fragment, false); handled || err != nil {
				if err != nil {
					return nil, err
				}
				continue
			}
		} else if name == "form" && tag && !closing && p.currentTagTokenWillEmit() {
			updateIgnoredStartBoundary()
			p.consumeIgnoredStartTag()
			continue
		}

		if mode == templateModeColumnGroup && (!tag || closing || (name != "col" && name != "template")) {
			switch {
			case p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction():
				child, err := p.parseNode(fragment)
				if err != nil {
					return nil, err
				}
				p.appendTemplateChild(fragment, child)
			case p.nextTokenIsIgnoredDoctype():
				if _, err := p.parseNode(fragment); err != nil {
					return nil, err
				}
			case tag && !closing:
				updateIgnoredStartBoundary()
				p.consumeIgnoredStartTag()
			case tag && closing:
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
			default:
				if whitespace := p.parseTemplateColumnWhitespace(fragment); whitespace != nil {
					p.appendTemplateChild(fragment, whitespace)
				} else if _, err := p.parseTextNode(fragment, p.pos, p.line, p.col); err != nil {
					return nil, err
				}
			}
			continue
		}

		if tag && !closing && ((mode == templateModeTableBody && (name == "caption" || name == "col" || name == "colgroup" || name == "tbody" || name == "tfoot" || name == "thead")) ||
			(mode == templateModeRow && (name == "caption" || name == "col" || name == "colgroup" || name == "tbody" || name == "tfoot" || name == "thead" || name == "tr"))) {
			updateIgnoredStartBoundary()
			p.consumeIgnoredStartTag()
			continue
		}
		if tag && !closing && name == "table" && mode != templateModeInitial && mode != templateModeBody {
			updateIgnoredStartBoundary()
			p.consumeIgnoredStartTag()
			continue
		}

		if mode == templateModeBody && tag && ((closing && isTableStructuralName(name)) || (!closing && isTableStructureStart(name))) {
			if closing {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
			} else {
				updateIgnoredStartBoundary()
				p.consumeIgnoredStartTag()
			}
			continue
		}
		var child *types.Node
		var err error
		ordinary := false
		if tag && !closing {
			switch mode {
			case templateModeTable:
				switch name {
				case "caption":
					child, err = p.parseTableLooseContainer(fragment, "caption", map[string]bool{"caption": true, "col": true, "colgroup": true, "tbody": true, "tfoot": true, "thead": true, "tr": true})
				case "colgroup":
					child, err = p.parseTableColgroup(fragment, false)
				case "col":
					child, err = p.parseTableColgroup(fragment, true)
				case "tbody", "thead", "tfoot":
					child, err = p.parseTableSection(fragment, false)
				case "tr", "td", "th":
					child, err = p.parseTableSection(fragment, true)
				default:
					ordinary = true
				}
			case templateModeColumnGroup:
				child, err = p.parseNode(fragment)
			case templateModeTableBody:
				switch name {
				case "tr":
					child, err = p.parseTableRow(fragment, false)
				case "td", "th":
					child, err = p.parseTableRow(fragment, true)
				default:
					ordinary = true
				}
			case templateModeRow:
				switch name {
				case "td", "th":
					child, err = p.parseTableCell(fragment)
				default:
					ordinary = true
				}
			default:
				ordinary = true
			}
		} else {
			ordinary = true
		}
		if ordinary {
			insertionParent := p.formattingInsertionParent(fragment, p.nextTokenReconstructsFormatting())
			child, err = p.parseNode(insertionParent)
			if err == nil && child != nil && insertionParent != fragment {
				p.appendFormattingChild(fragment, insertionParent, child)
				child = nil
			}
		}
		if err != nil {
			return nil, err
		}
		p.appendTemplateChild(fragment, child)
		if boundary, line, column, ok := p.templateElementBoundary(child); ok {
			lastBoundary, lastBoundaryLine, lastBoundaryColumn = boundary, line, column
		}
	}

	p.refreshOpenFormattingText()
	refreshNodeTextContent(fragment)
	// parse5 ends an unclosed template at the current token boundary. Preserve
	// the normal source-backed host location while keeping the fragment itself
	// locationless.
	if boundary, ok := p.templateEOFBoundary[template]; ok {
		template.ContentStart = contentStart
		template.ContentEnd = boundary
		template.EndPos = boundary
		template.EndLine = p.templateEOFBoundaryLine[template]
		template.EndColumn = p.templateEOFBoundaryColumn[template]
	} else {
		template.ContentStart = contentStart
		template.ContentEnd = lastBoundary
		template.EndPos = lastBoundary
		template.EndLine = lastBoundaryLine
		template.EndColumn = lastBoundaryColumn
	}
	return template, nil
}

func (p *HTMLParser) templateElementBoundary(node *types.Node) (int, int, int, bool) {
	if node == nil || node.Type != types.ElementNode {
		return 0, 0, 0, false
	}
	if node.StartLine != 0 {
		boundary := node.ContentEnd
		if p.isSelfClosingTag(node.Name) || node.ContentEnd >= node.EndPos {
			boundary = node.StartPos
		}
		line, column := p.coordinatesFromNodeStart(node, boundary)
		return boundary, line, column, true
	}
	for index := len(node.Children) - 1; index >= 0; index-- {
		if boundary, line, column, ok := p.templateElementBoundary(node.Children[index]); ok {
			return boundary, line, column, true
		}
	}
	return 0, 0, 0, false
}

func (p *HTMLParser) parseTemplateColumnWhitespace(parent *types.Node) *types.Node {
	start, line, column := p.pos, p.line, p.col
	var value strings.Builder
	for p.pos < len(p.content) {
		if p.peek() == '<' && p.startsMarkupToken() {
			break
		}
		preview := *p
		if decoded, consumed := preview.consumeNumericCharacterReference(); consumed {
			if !isHTMLWhitespaceString(string(decoded)) {
				break
			}
			decoded, _ = p.consumeNumericCharacterReference()
			value.WriteRune(decoded)
			continue
		}
		preview = *p
		if decoded, consumed := preview.consumeNamedCharacterReference(false); consumed {
			if !isHTMLWhitespaceString(decoded) {
				break
			}
			decoded, _ = p.consumeNamedCharacterReference(false)
			value.WriteString(decoded)
			continue
		}
		r, size := p.peekHTMLRune()
		if size == 0 || !isHTMLWhitespaceString(string(r)) {
			break
		}
		value.WriteRune(r)
		p.advanceRune(size)
	}
	if value.Len() == 0 {
		return nil
	}
	text := value.String()
	return &types.Node{
		Type: types.TextNode, Name: "#text", Value: text, TextContent: text, Parent: parent,
		StartPos: start, EndPos: p.pos, StartLine: line, StartColumn: column,
		EndLine: p.line, EndColumn: p.col,
	}
}

// templatePseudoTableAnythingElse identifies a token that a synthetic
// table-body/row frame cannot own because there is no real table element in
// scope. Return it to the template driver so adjusted foster insertion and
// active-formatting reconstruction happen at the inert fragment root.
func (p *HTMLParser) templatePseudoTableAnythingElse(node *types.Node) bool {
	if p.activeTable != nil || p.templateOwnerForNode(node) == nil || p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() || p.nextTokenIsIgnoredDoctype() {
		return false
	}
	if name, closing, tag := p.peekTagName(); tag {
		return closing || (name != "template" && name != "style" && name != "script" && !isTableStructuralName(name))
	}
	preview := *p
	parent := &types.Node{Type: types.ElementNode, Name: "#preview", Children: []*types.Node{}}
	text, err := preview.parseTextNode(parent, preview.pos, preview.line, preview.col)
	return err == nil && text != nil && !isHTMLWhitespaceString(text.Value)
}

func (p *HTMLParser) consumeTemplatePseudoAbsentEnd(node *types.Node) (bool, error) {
	if p.activeTable != nil || p.templateOwnerForNode(node) == nil {
		return false, nil
	}
	name, closing, tag := p.peekTagName()
	if !tag || !closing || name == "template" || name == "p" || name == "br" || name == "form" || name == "script" || isTableStructuralName(name) || isCoreAdoptionSubject(name) || !p.currentTagTokenWillEmit() {
		return false, nil
	}
	_, _, err := p.parseClosingTag()
	return true, err
}

func (p *HTMLParser) coordinatesFromNodeStart(node *types.Node, target int) (int, int) {
	preview := HTMLParser{
		content: p.content,
		pos:     node.StartPos,
		line:    node.StartLine,
		col:     node.StartColumn,
	}
	for preview.pos < target {
		_, size := preview.peekHTMLRune()
		if size == 0 || preview.pos+size > target {
			break
		}
		preview.advanceRune(size)
	}
	return preview.line, preview.col
}

func (p *HTMLParser) finishImplicitElement(node *types.Node, textContent string, contentStart int) {
	node.TextContent = textContent
	if node.StartLine == 0 {
		return
	}
	if owner := p.templateOwnerForNode(node); owner != nil && p.pos == len(p.content) {
		boundary, ok := p.templateEOFBoundary[owner]
		if !ok {
			boundary = node.StartPos
			p.templateEOFBoundary[owner] = boundary
			p.templateEOFBoundaryLine[owner] = node.StartLine
			p.templateEOFBoundaryColumn[owner] = node.StartColumn
		}
		node.ContentStart = contentStart
		node.ContentEnd = boundary
		node.EndPos = boundary
		node.EndLine = p.templateEOFBoundaryLine[owner]
		node.EndColumn = p.templateEOFBoundaryColumn[owner]
		return
	}
	if node == p.documentBody && p.documentBodyEndPos > 0 {
		node.ContentStart = contentStart
		node.ContentEnd = p.documentBodyContentEnd
		node.EndPos = p.documentBodyEndPos
		node.EndLine = p.documentBodyEndLine
		node.EndColumn = p.documentBodyEndColumn
		return
	}
	if node == p.documentHTML && p.documentHTMLEndPos > 0 {
		node.ContentStart = contentStart
		node.ContentEnd = p.documentHTMLContentEnd
		node.EndPos = p.documentHTMLEndPos
		node.EndLine = p.documentHTMLEndLine
		node.EndColumn = p.documentHTMLEndColumn
		return
	}
	node.ContentStart = contentStart
	node.ContentEnd = p.pos
	node.EndPos = p.pos
	node.EndLine = p.line
	node.EndColumn = p.col
}

func (p *HTMLParser) finishElementAt(node *types.Node, textContent string, contentStart, end, line, column int) {
	node.TextContent = textContent
	if node.StartLine == 0 {
		return
	}
	node.ContentStart = contentStart
	node.ContentEnd = end
	node.EndPos = end
	node.EndLine = line
	node.EndColumn = column
}

func syntheticParagraph(parent *types.Node) *types.Node {
	return &types.Node{
		Type: types.ElementNode, Name: "p", Attributes: make(map[string]string),
		Children: []*types.Node{}, Parent: parent,
	}
}

type explicitDocumentMode uint8

const (
	explicitBeforeHead explicitDocumentMode = iota
	explicitInHead
	explicitAfterHead
	explicitInBody
	explicitAfterBody
)

func syntheticDocumentWrapper(name string, parent *types.Node) *types.Node {
	return &types.Node{
		Type:                types.ElementNode,
		Name:                name,
		NamespaceURI:        htmlElementNamespace,
		Attributes:          make(map[string]string),
		AttributeOrder:      []string{},
		AttributeNamespaces: make(map[string]string),
		AttributeLocalNames: make(map[string]string),
		AttributePrefixes:   make(map[string]string),
		Children:            []*types.Node{},
		Parent:              parent,
	}
}

func appendDocumentChild(parent, child *types.Node) {
	if parent == nil || child == nil {
		return
	}
	child.Parent = parent
	if child.Type == types.TextNode && len(parent.Children) > 0 && parent.Children[len(parent.Children)-1].Type == types.TextNode {
		mergeTextNodes(parent.Children[len(parent.Children)-1], child)
		return
	}
	parent.Children = append(parent.Children, child)
}

func (p *HTMLParser) ensureDocumentHead(html *types.Node) *types.Node {
	if p.documentHead == nil {
		p.documentHead = syntheticDocumentWrapper("head", html)
		appendDocumentChild(html, p.documentHead)
	}
	return p.documentHead
}

func (p *HTMLParser) ensureDocumentBody(html *types.Node) *types.Node {
	if p.documentHead == nil {
		p.ensureDocumentHead(html)
	}
	if p.documentBody == nil {
		p.documentBody = syntheticDocumentWrapper("body", html)
		appendDocumentChild(html, p.documentBody)
	}
	p.headExited = true
	p.bodyContentStarted = true
	return p.documentBody
}

// parseLeadingDocumentWhitespace consumes only data-state character tokens
// whose decoded values are HTML whitespace. It stops before the first decoded
// non-whitespace token, preserving its raw source offset for reprocessing in
// the next insertion mode.
func (p *HTMLParser) parseLeadingDocumentWhitespace(parent *types.Node) *types.Node {
	// A prior insertion-mode pass may already have split an ignored whitespace
	// prefix from the first decoded nonspace reference. Leave that reference
	// untouched until the body-mode text parser consumes it and applies the
	// saved parse5 source boundary.
	if p.documentTextStartOverride >= p.pos {
		return nil
	}
	start, line, column := p.pos, p.line, p.col
	var value strings.Builder
	p.documentTextStartOverride = -1
	for p.pos < len(p.content) {
		if p.peek() == '<' && p.startsMarkupToken() {
			break
		}
		if p.peek() == 0 {
			p.advance()
			continue
		}
		preview := *p
		if decoded, consumed := preview.consumeNumericCharacterReference(); consumed {
			if !isHTMLWhitespaceString(string(decoded)) {
				p.documentTextStartOverride = preview.pos - 1
				p.documentTextStartLine = preview.line
				p.documentTextStartColumn = preview.col - 1
				break
			}
			decoded, _ = p.consumeNumericCharacterReference()
			value.WriteRune(decoded)
			continue
		}
		preview = *p
		if decoded, consumed := preview.consumeNamedCharacterReference(false); consumed {
			if !isHTMLWhitespaceString(decoded) {
				p.documentTextStartOverride = preview.pos - 1
				p.documentTextStartLine = preview.line
				p.documentTextStartColumn = preview.col - 1
				break
			}
			decoded, _ = p.consumeNamedCharacterReference(false)
			value.WriteString(decoded)
			continue
		}
		r, size := p.peekHTMLRune()
		if size == 0 || !isHTMLWhitespaceString(string(r)) {
			break
		}
		value.WriteRune(r)
		p.advanceRune(size)
	}
	if value.Len() == 0 {
		// A non-whitespace character reference at the beginning of the run is
		// parsed normally by the in-body text tokenizer and retains its whole
		// raw source range. The semicolon boundary override is only the parse5
		// convention for a nonspace reference following an ignored whitespace
		// prefix split across insertion modes.
		p.documentTextStartOverride = -1
		return nil
	}
	text := value.String()
	end, endLine, endColumn := p.pos, p.line, p.col
	if p.documentTextStartOverride >= 0 {
		end = p.documentTextStartOverride
		endLine = p.documentTextStartLine
		endColumn = p.documentTextStartColumn
	}
	return &types.Node{
		Type: types.TextNode, Name: "#text", Value: text, TextContent: text, Parent: parent,
		StartPos: start, EndPos: end, StartLine: line, StartColumn: column,
		EndLine: endLine, EndColumn: endColumn,
	}
}

func (p *HTMLParser) consumeDocumentDoctype(parent *types.Node) error {
	start, line, column := p.pos, p.line, p.col
	_, err := p.parseDoctype(parent, start, line, column)
	return err
}

func (p *HTMLParser) sourceCoordinatesAt(offset int) (int, int) {
	preview := HTMLParser{content: p.content, line: 1, col: 1}
	for preview.pos < offset {
		_, size := preview.peekHTMLRune()
		if size == 0 || preview.pos+size > offset {
			break
		}
		preview.advanceRune(size)
	}
	return preview.line, preview.col
}

// parse5/jsdom end an implicitly closed source-backed HEAD at the end of its
// last inserted child content (for example, immediately before a title end
// tag), while the containing HTML continues through EOF. Preserve that
// observable source-location convention for an omitted or discarded </head>.
func (p *HTMLParser) finishDocumentHeadAtLastContent(head *types.Node, text string, contentStart int) {
	end := contentStart
	if len(head.Children) > 0 {
		last := head.Children[len(head.Children)-1]
		if last.Type == types.ElementNode && last.ContentEnd > 0 {
			end = last.ContentEnd
		} else if last.EndPos > 0 {
			end = last.EndPos
		}
	}
	line, column := p.sourceCoordinatesAt(end)
	p.finishElementAt(head, text, contentStart, end, line, column)
}

func (p *HTMLParser) parseHeadNoscriptDisabled(node *types.Node, contentStart int) (*types.Node, error) {
	var text strings.Builder
	acceptedChildBoundary := 0
	acceptedChildLine := 0
	acceptedChildColumn := 0
	for p.pos < len(p.content) {
		if whitespace := p.parseTemplateColumnWhitespace(node); whitespace != nil {
			appendDocumentChild(node, whitespace)
			text.WriteString(whitespace.Value)
			continue
		}
		if p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() {
			child, err := p.parseNode(node)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(node, child)
			continue
		}
		if p.nextTokenIsIgnoredDoctype() {
			if err := p.consumeDocumentDoctype(node); err != nil {
				return nil, err
			}
			continue
		}

		name, closing, tag := p.peekTagName()
		if tag && closing && name == "noscript" {
			end := p.pos
			_, emitted, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if !emitted {
				p.finishImplicitElement(node, text.String(), contentStart)
				return node, nil
			}
			node.TextContent = text.String()
			node.ContentStart = contentStart
			node.ContentEnd = end
			node.EndPos = p.pos
			node.EndLine = p.line
			node.EndColumn = p.col
			return node, nil
		}
		if tag && closing {
			if name == "br" {
				p.finishElementAt(node, text.String(), contentStart, p.pos, p.line, p.col)
				return node, nil
			}
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if tag && !closing && name == "html" && p.currentTagTokenWillEmit() {
			if err := p.mergeDuplicateStartInto(p.documentHTML, "html"); err != nil {
				return nil, err
			}
			continue
		}
		if tag && !closing && (name == "head" || name == "noscript") && p.currentTagTokenWillEmit() {
			p.consumeIgnoredStartTag()
			continue
		}
		if tag && !closing && (name == "basefont" || name == "bgsound" || name == "link" || name == "meta" || name == "noframes" || name == "style") && p.currentTagTokenWillEmit() {
			child, err := p.parseNode(node)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(node, child)
			if child != nil && child.Type == types.ElementNode {
				text.WriteString(child.TextContent)
				acceptedChildBoundary = child.ContentEnd
				if p.isSelfClosingTag(child.Name) || child.ContentEnd >= child.EndPos {
					acceptedChildBoundary = child.StartPos
				}
				acceptedChildLine, acceptedChildColumn = p.sourceCoordinatesAt(acceptedChildBoundary)
			}
			continue
		}
		if tag && !p.currentTagTokenWillEmit() {
			child, err := p.parseNode(node)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(node, child)
			continue
		}

		// Anything else pops noscript and is reprocessed by the surrounding
		// in-head/after-head driver without consuming the triggering token.
		p.finishElementAt(node, text.String(), contentStart, p.pos, p.line, p.col)
		return node, nil
	}
	if acceptedChildBoundary > 0 {
		p.finishElementAt(node, text.String(), contentStart, acceptedChildBoundary, acceptedChildLine, acceptedChildColumn)
		return node, nil
	}
	p.finishImplicitElement(node, text.String(), contentStart)
	return node, nil
}

func (p *HTMLParser) parseExplicitDocumentHead(head *types.Node, contentStart int) (*types.Node, error) {
	p.documentHead = head
	var text strings.Builder
	for p.pos < len(p.content) {
		if whitespace := p.parseLeadingDocumentWhitespace(head); whitespace != nil {
			appendDocumentChild(head, whitespace)
			text.WriteString(whitespace.Value)
			continue
		}
		if p.pos >= len(p.content) {
			break
		}
		if p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() {
			child, err := p.parseNode(head)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(head, child)
			continue
		}
		if p.nextTokenIsIgnoredDoctype() {
			if err := p.consumeDocumentDoctype(head); err != nil {
				return nil, err
			}
			continue
		}
		name, closing, tag := p.peekTagName()
		if tag && closing && name != "head" && name != "body" && name != "html" && name != "br" {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if tag && closing && name == "head" {
			start := p.pos
			_, emitted, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if emitted {
				head.TextContent = text.String()
				head.ContentStart = contentStart
				head.ContentEnd = start
				head.EndPos = p.pos
				head.EndLine = p.line
				head.EndColumn = p.col
			} else {
				p.finishDocumentHeadAtLastContent(head, text.String(), contentStart)
			}
			p.headExited = true
			return head, nil
		}
		if tag && !closing && name == "head" && p.currentTagTokenWillEmit() {
			p.consumeIgnoredStartTag()
			continue
		}
		if tag && !closing && (isHeadOnlyElement(name) || name == "noscript") && name != "head" && p.currentTagTokenWillEmit() {
			child, err := p.parseNode(head)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(head, child)
			if child != nil && child.Type == types.ElementNode {
				text.WriteString(child.TextContent)
			}
			continue
		}
		if tag && !p.currentTagTokenWillEmit() {
			child, err := p.parseNode(head)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(head, child)
			continue
		}
		// Any other emitted token implicitly closes HEAD and is reprocessed by
		// the after-head insertion mode.
		p.finishElementAt(head, text.String(), contentStart, p.pos, p.line, p.col)
		p.headExited = true
		return head, nil
	}
	p.finishDocumentHeadAtLastContent(head, text.String(), contentStart)
	p.headExited = true
	return head, nil
}

func (p *HTMLParser) finalizeExplicitHTMLAtEOF(html *types.Node, contentStart int) *types.Node {
	p.ensureDocumentHead(html)
	if p.documentFrameset == nil {
		p.ensureDocumentBody(html)
	}
	refreshNodeTextContent(p.documentHead)
	if p.documentBody != nil {
		refreshNodeTextContent(p.documentBody)
	}
	refreshNodeTextContent(html)
	html.ContentStart = contentStart
	if p.documentHTMLEndPos > 0 {
		// A descendant recovery path may consume the explicit </html> while
		// keeping virtual descendants open for after-after-body reprocessing.
		// Their later content changes DOM/TextContent, but not the source-backed
		// HTML element's already established token range.
		html.ContentEnd = p.documentHTMLContentEnd
		html.EndPos = p.documentHTMLEndPos
		html.EndLine = p.documentHTMLEndLine
		html.EndColumn = p.documentHTMLEndColumn
		return html
	}
	html.ContentEnd = p.pos
	html.EndPos = p.pos
	html.EndLine = p.line
	html.EndColumn = p.col
	return html
}

func (p *HTMLParser) parseExplicitDocumentHTML(html *types.Node, contentStart int) (*types.Node, error) {
	p.documentHTML = html
	mode := explicitBeforeHead

	for p.pos < len(p.content) {
		// List/select recovery can consume a body end while still inside a
		// descendant frame and defer following after-body comments until that
		// frame returns. Materialize them as HTML children before routing the
		// next token so their source order is preserved.
		if len(p.deferredHTMLComments) > 0 && p.documentBody != nil {
			comments := p.deferredHTMLComments
			p.deferredHTMLComments = nil
			for _, comment := range comments {
				appendDocumentChild(html, comment)
			}
		}
		// A body/html end token can change the document insertion mode while
		// descendants remain logically open. Before routing the next token by
		// the wrapper mode, reprocess ordinary content through the deepest such
		// descendant. Comments and processing instructions retain their
		// after-body/after-after-body placement and therefore bypass this path.
		if pending := p.pendingDocumentEndTarget(); pending != nil &&
			!p.nextTokenIsComment() && !p.nextTokenIsProcessingInstruction() && !p.nextTokenIsIgnoredDoctype() {
			name, closing, tag := p.peekTagName()
			if tag && !closing && p.paragraphClosingStartTag(name) && p.currentTagTokenWillEmit() {
				pending = p.resolvePendingThroughParagraph()
				parent := pending
				if parent == nil {
					parent = p.ensureDocumentBody(html)
				}
				child, err := p.parseNode(parent)
				if err != nil {
					return nil, err
				}
				if child != nil {
					if pending != nil {
						p.appendToPendingDocumentEnd(pending, child)
					} else {
						appendDocumentChild(parent, child)
					}
				}
				continue
			}
			if tag && closing {
				if name == "body" || name == "html" {
					// The wrapper insertion mode consumes these without popping the
					// pending descendant identities.
				} else if resolved, err := p.resolvePendingThroughName(name, true); err != nil {
					return nil, err
				} else if resolved {
					continue
				} else {
					child, err := p.parseNode(pending)
					if err != nil {
						return nil, err
					}
					if child != nil {
						p.appendToPendingDocumentEnd(pending, child)
					}
					continue
				}
			} else {
				child, err := p.parseNode(pending)
				if err != nil {
					return nil, err
				}
				if child != nil {
					p.appendToPendingDocumentEnd(pending, child)
				}
				continue
			}
		}
		switch mode {
		case explicitBeforeHead:
			if whitespace := p.parseLeadingDocumentWhitespace(html); whitespace != nil {
				// Before-head whitespace character tokens are ignored.
				continue
			}
			if p.pos >= len(p.content) {
				break
			}
			if p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(html, child)
				continue
			}
			if p.nextTokenIsIgnoredDoctype() {
				if err := p.consumeDocumentDoctype(html); err != nil {
					return nil, err
				}
				continue
			}
			name, closing, tag := p.peekTagName()
			if tag && closing && name != "head" && name != "body" && name != "html" && name != "br" {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				continue
			}
			if tag && closing && name == "head" {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				continue
			}
			if tag && !closing && name == "html" && p.currentTagTokenWillEmit() {
				if err := p.mergeDuplicateStartInto(html, "html"); err != nil {
					return nil, err
				}
				continue
			}
			if tag && !closing && name == "head" && p.currentTagTokenWillEmit() {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				if child != nil {
					p.documentHead = child
					appendDocumentChild(html, child)
					mode = explicitAfterHead
				}
				continue
			}
			if tag && !closing && (isHeadOnlyElement(name) || name == "noscript") && name != "head" && p.currentTagTokenWillEmit() {
				head := p.ensureDocumentHead(html)
				mode = explicitInHead
				child, err := p.parseNode(head)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(head, child)
				continue
			}
			if tag && !p.currentTagTokenWillEmit() {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(html, child)
				continue
			}
			p.ensureDocumentHead(html)
			mode = explicitAfterHead
			continue

		case explicitInHead:
			head := p.ensureDocumentHead(html)
			if whitespace := p.parseLeadingDocumentWhitespace(head); whitespace != nil {
				appendDocumentChild(head, whitespace)
				continue
			}
			if p.pos >= len(p.content) {
				break
			}
			if p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() {
				child, err := p.parseNode(head)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(head, child)
				continue
			}
			if p.nextTokenIsIgnoredDoctype() {
				if err := p.consumeDocumentDoctype(head); err != nil {
					return nil, err
				}
				continue
			}
			name, closing, tag := p.peekTagName()
			if tag && closing && name != "head" && name != "body" && name != "html" && name != "br" {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				continue
			}
			if tag && closing && name == "head" {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				mode = explicitAfterHead
				p.headExited = true
				continue
			}
			if tag && !closing && name == "head" && p.currentTagTokenWillEmit() {
				p.consumeIgnoredStartTag()
				continue
			}
			if tag && !closing && (isHeadOnlyElement(name) || name == "noscript") && name != "head" && p.currentTagTokenWillEmit() {
				child, err := p.parseNode(head)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(head, child)
				continue
			}
			if tag && !p.currentTagTokenWillEmit() {
				child, err := p.parseNode(head)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(head, child)
				continue
			}
			mode = explicitAfterHead
			p.headExited = true
			continue

		case explicitAfterHead:
			if whitespace := p.parseLeadingDocumentWhitespace(html); whitespace != nil {
				appendDocumentChild(html, whitespace)
				continue
			}
			if p.pos >= len(p.content) {
				break
			}
			if p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(html, child)
				continue
			}
			if p.nextTokenIsIgnoredDoctype() {
				if err := p.consumeDocumentDoctype(html); err != nil {
					return nil, err
				}
				continue
			}
			name, closing, tag := p.peekTagName()
			if tag && !closing && name == "html" && p.currentTagTokenWillEmit() {
				if err := p.mergeDuplicateStartInto(html, "html"); err != nil {
					return nil, err
				}
				continue
			}
			if tag && !closing && name == "head" && p.currentTagTokenWillEmit() {
				p.consumeIgnoredStartTag()
				continue
			}
			if tag && !closing && isHeadOnlyElement(name) && name != "head" && p.currentTagTokenWillEmit() {
				head := p.ensureDocumentHead(html)
				// The after-head insertion mode temporarily reuses the retained
				// head pointer for head-only tokens. parse5 extends a source-backed
				// HEAD through any intervening after-head tokens up to the start of
				// the token routed back into HEAD; the routed token itself remains
				// outside that source range. Synthetic heads stay locationless.
				if head.StartLine != 0 {
					head.ContentEnd = p.pos
					head.EndPos = p.pos
					head.EndLine = p.line
					head.EndColumn = p.col
				}
				child, err := p.parseNode(head)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(head, child)
				continue
			}
			if tag && !closing && name == "body" && p.currentTagTokenWillEmit() {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				if child != nil {
					// A source BODY that reaches EOF without an emitted end tag has
					// only start-tag source location in parse5/DOM. Its descendants
					// still close at EOF. An explicit </body> ending at EOF retains
					// the full source-backed range.
					if p.pos == len(p.content) && child.StartLine != 0 && html.StartLine == 0 {
						explicitEnd := p.hasEmittedEndTagBetween("body", child.ContentStart, p.pos) ||
							p.hasEmittedEndTagBetween("html", child.ContentStart, p.pos)
						if !explicitEnd {
							child.ContentEnd = child.ContentStart
							child.EndPos = child.ContentStart
							child.EndLine = p.elementContentStartLine[child]
							child.EndColumn = p.elementContentStartColumn[child]
						}
					}
					p.documentBody = child
					appendDocumentChild(html, child)
					p.appendDeferredListComments(html, child)
					mode = explicitAfterBody
				}
				continue
			}
			if tag && !closing && name == "frameset" && p.currentTagTokenWillEmit() && p.documentBody == nil {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				if child != nil {
					p.documentFrameset = child
					appendDocumentChild(html, child)
					mode = explicitAfterBody
				}
				continue
			}
			if tag && !p.currentTagTokenWillEmit() {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(html, child)
				continue
			}
			p.ensureDocumentBody(html)
			mode = explicitInBody
			continue

		case explicitInBody:
			body := p.ensureDocumentBody(html)
			name, closing, tag := p.peekTagName()
			if tag && closing && name == "head" {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				continue
			}
			if tag && closing && name == "html" {
				mode = explicitAfterBody
				continue
			}
			if tag && closing && name == "body" {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
				mode = explicitAfterBody
				continue
			}
			if tag && !closing && name == "html" && p.currentTagTokenWillEmit() {
				if err := p.mergeDuplicateStartInto(html, "html"); err != nil {
					return nil, err
				}
				continue
			}
			if tag && !closing && name == "body" && p.currentTagTokenWillEmit() {
				if err := p.mergeDuplicateStartInto(body, "body"); err != nil {
					return nil, err
				}
				continue
			}
			if tag && !closing && name == "head" && p.currentTagTokenWillEmit() {
				p.consumeIgnoredStartTag()
				continue
			}
			if tag && !closing && (name == "frameset" || name == "frame") && p.currentTagTokenWillEmit() {
				p.consumeIgnoredStartTag()
				continue
			}
			if p.nextTokenIsIgnoredDoctype() {
				if err := p.consumeDocumentDoctype(body); err != nil {
					return nil, err
				}
				continue
			}
			if err := p.parseDocumentBodyToken(body); err != nil {
				return nil, err
			}

		case explicitAfterBody:
			if p.pos >= len(p.content) {
				break
			}
			if p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() {
				child, err := p.parseNode(html)
				if err != nil {
					return nil, err
				}
				appendDocumentChild(html, child)
				continue
			}
			if p.nextTokenIsIgnoredDoctype() {
				if err := p.consumeDocumentDoctype(html); err != nil {
					return nil, err
				}
				continue
			}
			name, closing, tag := p.peekTagName()
			if tag && closing && name == "html" {
				start := p.pos
				_, emitted, err := p.parseClosingTag()
				if err != nil {
					return nil, err
				}
				if emitted {
					p.ensureDocumentHead(html)
					if p.documentFrameset == nil {
						p.ensureDocumentBody(html)
					}
					refreshNodeTextContent(p.documentHead)
					if p.documentBody != nil {
						refreshNodeTextContent(p.documentBody)
					}
					refreshNodeTextContent(html)
					html.ContentStart = contentStart
					html.ContentEnd = start
					html.EndPos = p.pos
					html.EndLine = p.line
					html.EndColumn = p.col
					p.documentHTMLContentEnd = start
					p.documentHTMLEndPos = p.pos
					p.documentHTMLEndLine = p.line
					p.documentHTMLEndColumn = p.col
					p.documentEndRecovery = true
					p.explicitAfterAfterBody = true
					return html, nil
				}
				break
			}
			if tag && !closing && name == "html" && p.currentTagTokenWillEmit() {
				if err := p.mergeDuplicateStartInto(html, "html"); err != nil {
					return nil, err
				}
				continue
			}
			if tag && !closing && name == "body" && p.currentTagTokenWillEmit() {
				if err := p.mergeDuplicateStartInto(p.ensureDocumentBody(html), "body"); err != nil {
					return nil, err
				}
				continue
			}
			if p.documentFrameset != nil {
				if tag && !closing && name == "noframes" && p.currentTagTokenWillEmit() {
					head := p.ensureDocumentHead(html)
					child, err := p.parseNode(head)
					if err != nil {
						return nil, err
					}
					appendDocumentChild(head, child)
					continue
				}
				// The after-frameset insertion mode ignores ordinary tokens.
				// Whitespace is retained at the HTML level; comments/PIs and the
				// closing-html cases were already handled above.
				if whitespace := p.parseLeadingDocumentWhitespace(html); whitespace != nil {
					appendDocumentChild(html, whitespace)
					continue
				}
				if tag {
					if closing {
						if _, _, err := p.parseClosingTag(); err != nil {
							return nil, err
						}
					} else {
						p.consumeIgnoredStartTag()
					}
					continue
				}
				if _, err := p.parseTextNode(html, p.pos, p.line, p.col); err != nil {
					return nil, err
				}
				continue
			}
			// Whitespace and all ordinary tokens are processed using in-body
			// rules; comments/PIs above remain children of HTML.
			mode = explicitInBody
			continue
		}
	}

	return p.finalizeExplicitHTMLAtEOF(html, contentStart), nil
}

// parseDocumentBodyToken processes one token at the top level of a synthetic
// BODY. Source-backed BODY elements normally run the same rules from the
// recursive parseElement loop; a synthetic wrapper has no start tag/frame, so
// the document insertion-mode driver must preserve the identity-aware token
// handling that the former source-root loop provided.
func (p *HTMLParser) parseDocumentBodyToken(body *types.Node) error {
	if len(p.adoptionSplit) > 0 {
		p.appendAllDeferredAdoption(body)
	}
	if name, closing, ok := p.peekTagName(); ok && closing && p.adoptionIgnoredEnds[name] > 0 && p.activeFormattingEntryByName(name) == nil {
		start := p.pos
		if _, _, err := p.parseClosingTag(); err != nil {
			return err
		}
		p.extendPreviousTextRange(body, start)
		p.adoptionIgnoredEnds[name]--
		return nil
	}
	if consumed, err := p.consumeForeignIgnoredEnd(body); consumed || err != nil {
		return err
	}
	if name, closing, ok := p.peekTagName(); ok && closing && p.foreignIgnoredHTMLEnds[name] > 0 {
		start := p.pos
		if _, emitted, err := p.parseClosingTag(); err != nil {
			return err
		} else if !emitted {
			p.extendPreviousTextRange(body, start)
		}
		p.foreignIgnoredHTMLEnds[name]--
		return nil
	}
	if handled, err := p.handleForeignDocumentStart(); handled || err != nil {
		return err
	}
	if handled, err := p.handleFormToken(body, false); handled || err != nil {
		return err
	}
	if name, closing, ok := p.peekTagName(); ok && !closing && isTableStructureStart(name) && p.currentTagTokenWillEmit() {
		p.consumeIgnoredStartTag()
		return nil
	}
	if name, closing, ok := p.peekTagName(); ok && closing && isFormattingElement(name) {
		if isCoreAdoptionSubject(name) {
			if entry := p.activeFormattingEntryByName(name); entry == nil || entry.current == nil {
				start := p.pos
				if _, _, err := p.parseClosingTag(); err != nil {
					return err
				}
				if entry != nil {
					p.removeActiveFormattingEntry(entry)
				}
				p.extendPreviousTextRange(body, start)
				return nil
			}
		}
		if entry := p.activeFormattingEntryByName(name); entry != nil {
			if !p.openFormatting[entry.current] {
				start := p.pos
				if _, _, err := p.parseClosingTag(); err != nil {
					return err
				}
				p.extendPreviousTextRange(body, start)
				p.removeActiveFormattingEntry(entry)
				return nil
			}
			tokenStart := p.pos
			if _, _, err := p.parseClosingTag(); err != nil {
				return err
			}
			p.refreshFormattingChain(entry.current)
			if entry.current.StartLine != 0 {
				entry.current.ContentEnd = tokenStart
				entry.current.EndPos = p.pos
				entry.current.EndLine = p.line
				entry.current.EndColumn = p.col
			}
			p.removeActiveFormattingEntry(entry)
			return nil
		}
	}
	// The in-body "any other end tag" rule ignores a complete end tag when
	// no matching ordinary element is in scope. All identity-aware formatting,
	// form, list, select, and foreign consumers above get first refusal.
	if name, closing, ok := p.peekTagName(); ok && closing && name != "p" && name != "br" &&
		!p.isBogusEndTagOpen() && !p.isMissingEndTagName() && p.currentTagTokenWillEmit() {
		if _, _, err := p.parseClosingTag(); err != nil {
			return err
		}
		return nil
	}
	if action, err := p.handleSpecialFormattingStart(body, "", 0); err != nil {
		return err
	} else if action == adoptionContinue {
		return nil
	}
	processingInstruction := p.nextTokenIsProcessingInstruction()
	insertionParent := body
	if !processingInstruction {
		insertionParent = p.formattingInsertionParent(body, p.nextTokenReconstructsFormatting())
	}
	child, err := p.parseNode(insertionParent)
	if err != nil {
		return err
	}
	if child == nil {
		return nil
	}
	if child.Type == types.TextNode && p.documentTextStartOverride >= 0 {
		child.StartPos = p.documentTextStartOverride
		child.StartLine = p.documentTextStartLine
		child.StartColumn = p.documentTextStartColumn
		p.documentTextStartOverride = -1
	}
	if insertionParent != body {
		p.appendFormattingChild(body, insertionParent, child)
		if !p.isSelfClosingTag(child.Name) {
			return nil
		}
		return nil
	}
	appendDocumentChild(body, child)
	p.appendDeferredTableFoster(body, child)
	p.appendDeferredListComments(body, child)
	p.appendDeferredAdoption(body, child)
	p.appendDeferredFormSiblings(body, child)
	return nil
}

func isTableStructureStart(name string) bool {
	switch name {
	case "caption", "col", "colgroup", "tbody", "td", "tfoot", "th", "thead", "tr":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) hasEmittedEndTagBetween(name string, start, end int) bool {
	if start < 0 {
		start = 0
	}
	if end > len(p.content) {
		end = len(p.content)
	}
	for offset := start; offset+2 < end; {
		relative := strings.Index(p.content[offset:end], "</")
		if relative < 0 {
			return false
		}
		offset += relative
		preview := *p
		preview.pos = offset
		preview.tagPreviewPos = -1
		preview.tagPreviewEmissionKnown = false
		candidate, closing, ok := preview.peekTagName()
		if ok && closing && candidate == name && preview.currentTagTokenWillEmit() {
			return true
		}
		offset += 2
	}
	return false
}

// parseDocumentFramesetElement implements the bounded in-frameset insertion
// mode needed by whole-document parsing. Ordinary character/start/end tokens
// are ignored; whitespace, comments/PIs, nested framesets, frame, and noframes
// retain their browser insertion parents.
func (p *HTMLParser) parseDocumentFramesetElement(frameset *types.Node, contentStart int) (*types.Node, error) {
	var text strings.Builder
	for p.pos < len(p.content) {
		if whitespace := p.parseLeadingDocumentWhitespace(frameset); whitespace != nil {
			appendDocumentChild(frameset, whitespace)
			text.WriteString(whitespace.Value)
			continue
		}
		if p.pos >= len(p.content) {
			break
		}
		if p.nextTokenIsComment() || p.nextTokenIsProcessingInstruction() {
			child, err := p.parseNode(frameset)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(frameset, child)
			continue
		}
		if p.nextTokenIsIgnoredDoctype() {
			if err := p.consumeDocumentDoctype(frameset); err != nil {
				return nil, err
			}
			continue
		}
		name, closing, tag := p.peekTagName()
		if tag && closing && name == "frameset" {
			end := p.pos
			_, emitted, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if emitted {
				p.finishElementAt(frameset, text.String(), contentStart, end, p.line, p.col)
				frameset.EndPos = p.pos
				frameset.EndLine = p.line
				frameset.EndColumn = p.col
				return frameset, nil
			}
			break
		}
		if tag && !closing && (name == "frameset" || name == "frame" || name == "noframes") && p.currentTagTokenWillEmit() {
			child, err := p.parseNode(frameset)
			if err != nil {
				return nil, err
			}
			appendDocumentChild(frameset, child)
			if child != nil && child.Type == types.ElementNode {
				text.WriteString(child.TextContent)
			}
			continue
		}
		if tag {
			if closing {
				if _, _, err := p.parseClosingTag(); err != nil {
					return nil, err
				}
			} else {
				p.consumeIgnoredStartTag()
			}
			continue
		}
		if _, err := p.parseTextNode(frameset, p.pos, p.line, p.col); err != nil {
			return nil, err
		}
	}
	p.finishImplicitElement(frameset, text.String(), contentStart)
	return frameset, nil
}

func isHeadOnlyElement(name string) bool {
	switch name {
	case "base", "basefont", "bgsound", "head", "link", "meta", "noframes", "script", "style", "template", "title":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) enterElementContext(node *types.Node) func() {
	formattingMarker := isFormattingMarkerElement(node.Name)
	previousFormattingCounts := p.formattingCounts
	previousFormattingFamilies := p.formattingFamilies
	previousFormattingByName := p.formattingByName
	previousFormattingMarkerStart := p.formattingMarkerStart
	if formattingMarker {
		p.activeFormatting = append(p.activeFormatting, &formattingEntry{marker: true})
		p.formattingMarkerStart = len(p.activeFormatting)
		p.formattingCounts = make(map[string]int)
		p.formattingFamilies = make(map[string][]*formattingEntry)
		p.formattingByName = make(map[string][]*formattingEntry)
	}
	previousParagraph := p.activeParagraph
	previousAncestors := p.activeParagraphAncestors
	previousButton := p.activeButton
	previousHeadDepth := p.headDepth
	previousLIStart := p.activeLIStart
	previousDefinitionStart := p.activeDefinitionStart
	previousLIEnd := p.activeLIEnd
	previousDDEnd := p.activeDDEnd
	previousDTEnd := p.activeDTEnd
	previousSelect := p.activeSelect
	previousOptionStart := p.activeOptionStart
	previousOptgroupStart := p.activeOptgroupStart
	previousOptionEnd := p.activeOptionEnd
	previousOptgroupEnd := p.activeOptgroupEnd
	previousSelectImpliedEnd := p.activeSelectImpliedEnd
	previousSelectOptionImplied := p.activeSelectOptionImplied
	previousSelectNames := p.activeSelectNames
	previousSelectAllNames := p.activeSelectAllNames
	previousSelectBlockedByTable := p.selectBlockedByTable
	var previousSelectNameNode *types.Node
	previousSelectNamePresent := false
	var previousSelectAllNameNode *types.Node
	previousSelectAllNamePresent := false
	selectGenericScopeBoundary := false

	if node.Name == "head" {
		p.headDepth++
	} else if node.Name == "body" {
		p.documentBody = node
		p.headExited = true
		p.bodyContentStarted = true
	} else if node.Name != "html" && p.headDepth > 0 && !p.headExited && !isHeadOnlyElement(node.Name) && (node.Name != "noscript" || node.Parent != p.documentHead) {
		p.headExited = true
		p.bodyContentStarted = true
	} else if node.Name != "html" && p.headDepth == 0 && !isHeadOnlyElement(node.Name) {
		p.bodyContentStarted = true
	}
	if node.Name == "html" {
		p.documentHTML = node
	}

	if isListStartScanStopper(node.Name) {
		p.activeLIStart = nil
		p.activeDefinitionStart = nil
	}
	if isOrdinaryScopeBoundary(node.Name) {
		p.activeDDEnd = nil
		p.activeDTEnd = nil
	}
	if isListItemScopeBoundary(node.Name) {
		p.activeLIEnd = nil
	}
	// Customizable selects otherwise use the normal in-body rules. Keep two
	// views of option/group state: start-tag implied-end scans stop at the
	// first non-implied element, while matching end tags stop at an HTML
	// special element. Select itself is an ordinary-scope boundary.
	if !isImpliedEndTagName(node.Name) && node.Name != "select" {
		p.activeOptionStart = nil
		p.activeOptgroupStart = nil
		p.activeSelectImpliedEnd = nil
		p.activeSelectOptionImplied = nil
	}
	if isHTMLSpecialElement(node.Name) && node.Name != "option" && node.Name != "optgroup" && node.Name != "search" {
		p.activeOptionEnd = nil
		p.activeOptgroupEnd = nil
	}
	if isOrdinaryScopeBoundary(node.Name) {
		p.activeSelect = nil
	}
	if node.Name == "table" && previousSelect != nil {
		p.selectBlockedByTable = true
	}
	if p.activeSelect != nil && isImpliedEndTagName(node.Name) {
		p.activeSelectImpliedEnd = node
		if node.Name != "optgroup" {
			p.activeSelectOptionImplied = node
		}
	}
	switch node.Name {
	case "select":
		p.activeSelect = node
		p.activeOptionStart = nil
		p.activeOptgroupStart = nil
		p.activeOptionEnd = nil
		p.activeOptgroupEnd = nil
		p.activeSelectImpliedEnd = nil
		p.activeSelectOptionImplied = nil
		p.activeSelectNames = map[string]*types.Node{"select": node}
		p.activeSelectAllNames = map[string]*types.Node{"select": node}
	case "option":
		p.activeOptionStart = node
		p.activeOptionEnd = node
	case "optgroup":
		p.activeOptgroupStart = node
		p.activeOptgroupEnd = node
	}
	if p.activeSelect != nil && node.Name != "select" {
		selectGenericScopeBoundary = isHTMLSpecialElement(node.Name) && node.Name != "search"
		if selectGenericScopeBoundary {
			p.activeSelectNames = make(map[string]*types.Node)
		}
		if p.activeSelectNames == nil {
			p.activeSelectNames = make(map[string]*types.Node)
		}
		previousSelectNameNode, previousSelectNamePresent = p.activeSelectNames[node.Name]
		p.activeSelectNames[node.Name] = node
		if p.activeSelectAllNames == nil {
			p.activeSelectAllNames = make(map[string]*types.Node)
		}
		previousSelectAllNameNode, previousSelectAllNamePresent = p.activeSelectAllNames[node.Name]
		p.activeSelectAllNames[node.Name] = node
	}
	switch node.Name {
	case "li":
		p.activeLIStart = node
		p.activeLIEnd = node
	case "dd":
		p.activeDefinitionStart = node
		p.activeDDEnd = node
	case "dt":
		p.activeDefinitionStart = node
		p.activeDTEnd = node
	}

	if isParagraphButtonScopeBoundary(node.Name) {
		p.activeParagraph = nil
		p.activeParagraphAncestors = nil
		p.activeButton = nil
	}
	if node.Name == "button" && node.NamespaceURI == htmlElementNamespace {
		p.activeButton = node
	}
	if node.Name == "p" {
		p.activeParagraph = node
		p.activeParagraphAncestors = make(map[string]struct{})
		for ancestor := node.Parent; ancestor != nil; ancestor = ancestor.Parent {
			p.activeParagraphAncestors[ancestor.Name] = struct{}{}
			if isParagraphAncestorScopeBoundary(ancestor.Name) {
				break
			}
		}
	}

	return func() {
		if formattingMarker {
			p.clearActiveFormattingToMarker()
			p.formattingCounts = previousFormattingCounts
			p.formattingFamilies = previousFormattingFamilies
			p.formattingByName = previousFormattingByName
			p.formattingMarkerStart = previousFormattingMarkerStart
		}
		if node.Name == "html" {
			p.bodyContentStarted = true
		}
		p.activeParagraph = previousParagraph
		p.activeParagraphAncestors = previousAncestors
		p.activeButton = previousButton
		p.headDepth = previousHeadDepth
		p.activeLIStart = previousLIStart
		p.activeDefinitionStart = previousDefinitionStart
		p.activeLIEnd = previousLIEnd
		p.activeDDEnd = previousDDEnd
		p.activeDTEnd = previousDTEnd
		p.activeSelect = previousSelect
		p.activeOptionStart = previousOptionStart
		p.activeOptgroupStart = previousOptgroupStart
		p.activeOptionEnd = previousOptionEnd
		p.activeOptgroupEnd = previousOptgroupEnd
		p.activeSelectImpliedEnd = previousSelectImpliedEnd
		p.activeSelectOptionImplied = previousSelectOptionImplied
		if p.activeSelectNames != nil && node.Name != "select" && !selectGenericScopeBoundary {
			if previousSelectNamePresent {
				p.activeSelectNames[node.Name] = previousSelectNameNode
			} else {
				delete(p.activeSelectNames, node.Name)
			}
		}
		if p.activeSelectAllNames != nil && node.Name != "select" {
			if previousSelectAllNamePresent {
				p.activeSelectAllNames[node.Name] = previousSelectAllNameNode
			} else {
				delete(p.activeSelectAllNames, node.Name)
			}
		}
		p.activeSelectNames = previousSelectNames
		p.activeSelectAllNames = previousSelectAllNames
		p.selectBlockedByTable = previousSelectBlockedByTable
	}
}

func isImpliedEndTagName(name string) bool {
	switch name {
	case "dd", "dt", "li", "option", "optgroup", "p", "rb", "rp", "rt", "rtc":
		return true
	default:
		return false
	}
}

func isFormattingElement(name string) bool {
	switch name {
	case "a", "b", "big", "code", "em", "font", "i", "nobr", "s", "small", "strike", "strong", "tt", "u":
		return true
	default:
		return false
	}
}

func isCoreAdoptionSubject(name string) bool {
	switch name {
	case "a", "b", "big", "code", "em", "font", "i", "nobr", "s", "small", "strike", "strong", "tt", "u":
		return true
	default:
		return false
	}
}

type adoptionAction uint8

const (
	adoptionNone adoptionAction = iota
	adoptionContinue
	adoptionReturn
)

func (p *HTMLParser) consumeAdoptionEnd(consume bool) (bool, error) {
	if !consume {
		return true, nil
	}
	_, emitted, err := p.parseClosingTag()
	return emitted, err
}

func (p *HTMLParser) resetAdoptionUnwindCache() {
	p.adoptionUnwindPos = -1
	p.adoptionUnwindFormatting = nil
	p.adoptionUnwindBlock = nil
	p.adoptionUnwindInScope = false
}

func (p *HTMLParser) handleSpecialFormattingStart(current *types.Node, text string, contentStart int) (adoptionAction, error) {
	name, closing, ok := p.peekTagName()
	if !ok || closing || (name != "a" && name != "nobr") || !p.currentTagTokenWillEmit() {
		return adoptionNone, nil
	}
	firstPass := p.specialFormattingStartPos != p.pos || p.specialFormattingStartName != name
	if firstPass {
		p.specialFormattingStartPos = p.pos
		p.specialFormattingStartName = name
	}
	if name == "nobr" && firstPass {
		p.formattingInsertionParent(current, true)
	}
	entry := p.activeFormattingEntryByName(name)
	if entry == nil {
		return adoptionNone, nil
	}
	// A formatting entry can remain active after its source element has been
	// popped by a block.  An anchor start must still remove that exact old
	// entry before the normal reconstruction step.  For nobr the mandatory
	// reconstruction above gives the entry a current virtual-open clone.
	if entry.current == nil {
		if name == "a" {
			p.removeActiveFormattingEntry(entry)
			p.resetAdoptionUnwindCache()
		}
		return adoptionNone, nil
	}
	// Recursive source parents are not always the parser's effective open
	// formatting stack.  After block unwinding, reconstructed formatting
	// elements are DOM children of current while remaining virtually open.
	// Fake adoption must pop through the subject on that virtual stack, retain
	// later AFE entries for reconstruction, and remove only the old subject.
	virtualContainer := current
	if p.documentBody != nil && (current == p.documentHTML || current == p.documentRoot) {
		virtualContainer = p.documentBody
	}
	if p.openFormatting[entry.current] && p.formattingContainer[entry.current] == virtualContainer {
		p.closeVirtualFormattingThrough(entry, virtualContainer)
		return adoptionNone, nil
	}
	if name == "nobr" && !formattingInOrdinaryScope(entry.current, current) {
		return adoptionNone, nil
	}
	old := entry
	p.resetAdoptionUnwindCache()
	action, err := p.handleCoreAdoption(current, name, text, contentStart, false)
	if err != nil {
		return adoptionNone, err
	}
	if name == "a" && action != adoptionReturn {
		for _, candidate := range p.activeFormatting {
			if candidate == old {
				if old.current != nil && old.current.StartLine != 0 && !formattingInOrdinaryScope(old.current, current) {
					old.current.ContentEnd = p.pos
					old.current.EndPos = p.pos
					old.current.EndLine = p.line
					old.current.EndColumn = p.col
					p.adoptionReturnAtSplit[old.current] = true
				}
				p.removeActiveFormattingEntry(old)
				break
			}
		}
	}
	p.resetAdoptionUnwindCache()
	return action, nil
}

func (p *HTMLParser) closeVirtualFormattingThrough(subject *formattingEntry, container *types.Node) {
	if subject == nil || subject.current == nil {
		return
	}
	subjectNode := subject.current
	for index := len(p.formattingOpenStack) - 1; index >= 0; index-- {
		node := p.formattingOpenStack[index]
		if !p.openFormatting[node] || p.formattingContainer[node] != container {
			continue
		}
		p.refreshFormattingText(node)
		if node.StartLine != 0 {
			node.ContentEnd = p.pos
			node.EndPos = p.pos
			node.EndLine = p.line
			node.EndColumn = p.col
		}
		p.setFormattingOpen(node, false)
		if node == subjectNode {
			break
		}
		if entry := p.formattingByNode[node]; entry != nil {
			delete(p.formattingByNode, node)
			entry.current = nil
		}
	}
	p.removeActiveFormattingEntry(subject)
	p.resetAdoptionUnwindCache()
}

func (p *HTMLParser) handleCoreAdoption(current *types.Node, name, text string, contentStart int, consume bool) (adoptionAction, error) {
	entry := p.activeFormattingEntryByName(name)
	if entry == nil || entry.current == nil {
		if entry != nil {
			p.removeActiveFormattingEntry(entry)
		}
		if current.Name == name {
			return adoptionNone, nil
		}
		if target := adoptionGenericEndTarget(current, name); target != nil {
			closeStart := p.pos
			closeLine, closeColumn := p.line, p.col
			if emitted, err := p.consumeAdoptionEnd(consume); err != nil {
				return adoptionNone, err
			} else if !emitted {
				p.extendPreviousTextRange(current, closeStart)
				return adoptionContinue, nil
			}
			for node := current; node != nil && node != target; node = node.Parent {
				node.ContentEnd = closeStart
				node.EndPos = closeStart
				node.EndLine = closeLine
				node.EndColumn = closeColumn
				p.adoptionClosed[node] = true
				p.adoptionIgnoredEnds[node.Name]++
			}
			refreshTreeText(target)
			target.ContentEnd = closeStart
			target.EndPos = p.pos
			target.EndLine = p.line
			target.EndColumn = p.col
			p.adoptionClosed[target] = true
			return adoptionReturn, nil
		}
		start := p.pos
		if _, err := p.consumeAdoptionEnd(consume); err != nil {
			return adoptionNone, err
		}
		p.extendPreviousTextRange(current, start)
		if nearestOpenElementByName(current.Parent, name) != nil {
			p.recoverOpenElementsAtEOF = true
		}
		return adoptionContinue, nil
	}
	formatting := entry.current
	if !p.openFormatting[formatting] {
		start := p.pos
		if _, err := p.consumeAdoptionEnd(consume); err != nil {
			return adoptionNone, err
		}
		p.extendPreviousTextRange(current, start)
		p.removeActiveFormatting(formatting)
		return adoptionContinue, nil
	}
	if p.formattingContainer[formatting] == current && p.lastOpenFormattingNode() == formatting {
		documentEndClose := p.formattingDocumentEnd
		contentEnd := p.pos
		if emitted, err := p.consumeAdoptionEnd(consume); err != nil {
			return adoptionNone, err
		} else if !emitted {
			p.extendPreviousTextRange(formatting, contentEnd)
		}
		p.refreshFormattingChain(formatting)
		if formatting.StartLine != 0 {
			formatting.ContentEnd = contentEnd
			formatting.EndPos = p.pos
			formatting.EndLine = p.line
			formatting.EndColumn = p.col
		}
		p.removeActiveFormatting(formatting)
		if formatting.StartLine == 0 {
			formatting.StartPos, formatting.ContentStart, formatting.ContentEnd, formatting.EndPos = 0, 0, 0, 0
			formatting.StartColumn, formatting.EndLine, formatting.EndColumn = 0, 0, 0
		}
		if !documentEndClose {
			p.recoverOpenElementsAtEOF = true
			p.adoptionEOFBoundary = current
		}
		return adoptionContinue, nil
	}
	if p.formattingContainer[formatting] == current {
		documentEndClose := p.formattingDocumentEnd
		closeStart := p.pos
		if emitted, err := p.consumeAdoptionEnd(consume); err != nil {
			return adoptionNone, err
		} else if !emitted {
			p.extendPreviousTextRange(formatting, closeStart)
		}
		for node := p.lastOpenFormattingNode(); node != nil && node != formatting; node = node.Parent {
			entry := p.formattingByNode[node]
			if entry == nil {
				continue
			}
			p.refreshFormattingText(node)
			node.ContentEnd = closeStart
			node.EndPos = closeStart
			node.EndLine = p.line
			node.EndColumn = p.col
			p.setFormattingOpen(node, false)
			delete(p.formattingByNode, node)
			entry.current = nil
		}
		refreshTreeText(formatting)
		if formatting.StartLine != 0 {
			formatting.ContentEnd = closeStart
			formatting.EndPos = p.pos
			formatting.EndLine = p.line
			formatting.EndColumn = p.col
		}
		p.removeActiveFormatting(formatting)
		if !documentEndClose {
			p.recoverOpenElementsAtEOF = true
		}
		return adoptionContinue, nil
	}
	if !p.cachedAdoptionInScope(formatting, current) {
		start := p.pos
		if _, err := p.consumeAdoptionEnd(consume); err != nil {
			return adoptionNone, err
		}
		p.extendPreviousTextRange(current, start)
		return adoptionContinue, nil
	}
	if current == formatting {
		contentEnd := p.pos
		if emitted, err := p.consumeAdoptionEnd(consume); err != nil {
			return adoptionNone, err
		} else if !emitted {
			p.extendPreviousTextRange(current, contentEnd)
		}
		current.TextContent = text
		if current.StartLine != 0 {
			current.ContentStart = contentStart
			current.ContentEnd = contentEnd
			current.EndPos = p.pos
			current.EndLine = p.line
			current.EndColumn = p.col
		}
		p.removeActiveFormatting(current)
		return adoptionReturn, nil
	}
	if entry.current != entry.template && nodeWithin(entry.current, current) {
		contentEnd := p.pos
		if emitted, err := p.consumeAdoptionEnd(consume); err != nil {
			return adoptionNone, err
		} else if !emitted {
			p.extendPreviousTextRange(entry.current, contentEnd)
		}
		p.refreshFormattingChain(entry.current)
		if entry.current.StartLine != 0 {
			entry.current.ContentEnd = contentEnd
			entry.current.EndPos = p.pos
			entry.current.EndLine = p.line
			entry.current.EndColumn = p.col
		}
		p.removeActiveFormatting(entry.current)
		return adoptionContinue, nil
	}

	block := p.cachedAdoptionFurthestBlock(formatting, current)
	if block == nil {
		if current != formatting {
			p.adoptionIgnoredEnds[current.Name]++
		}
		p.finishImplicitElement(current, text, contentStart)
		return adoptionReturn, nil
	}
	if block != current {
		p.finishImplicitElement(current, text, contentStart)
		return adoptionReturn, nil
	}

	closeStart := p.pos
	closeLine, closeColumn := p.line, p.col
	if emitted, err := p.consumeAdoptionEnd(consume); err != nil {
		return adoptionNone, err
	} else if !emitted {
		p.extendIncompleteAdoptionAtEOF(block, closeStart)
		p.recoverOpenElementsAtEOF = true
		p.finishImplicitElement(block, block.TextContent, block.ContentStart)
		return adoptionReturn, nil
	}
	wrappers := p.adoptionInnerFormatting(formatting, block, closeStart, closeLine, closeColumn)
	oldAncestors := make([]*types.Node, 0, 4)
	for current := block.Parent; current != nil; current = current.Parent {
		oldAncestors = append(oldAncestors, current)
		if current == formatting {
			break
		}
	}
	p.detachDeferredAdoption(block)
	detachNodeShallow(block)
	p.adoptionPendingEnds[block] = adoptionSpecialDescendants(block)
	p.adoptionWrapSpecialContents(block, formatting, nil, closeStart, closeLine, closeColumn, 8)
	for _, current := range oldAncestors {
		delete(p.formattingText, current)
		refreshNodeTextContent(current)
	}
	for current := formatting; current != nil; current = current.Parent {
		if current.Type == types.DocumentNode {
			break
		}
		delete(p.formattingText, current)
		refreshNodeTextContent(current)
	}
	refreshTreeText(formatting)
	for node := block.Parent; node != nil && node != formatting; node = node.Parent {
		refreshNodeTextContent(node)
	}
	refreshNodeTextContent(block)
	if formatting.StartLine != 0 {
		formatting.ContentEnd = closeStart
		formatting.EndPos = p.pos
		formatting.EndLine = p.line
		formatting.EndColumn = p.col
	}
	p.removeActiveFormatting(formatting)
	if formatting.StartLine != 0 {
		p.adoptionReturnAtSplit[formatting] = true
	}
	splitNode := block
	if len(wrappers) > 0 {
		var outer *types.Node
		var container *types.Node
		clones := make([]*types.Node, 0, len(wrappers))
		for _, template := range wrappers {
			clone := cloneFormattingNode(template, container)
			clone.StartPos, clone.StartLine, clone.StartColumn = 0, 0, 0
			clone.ContentStart = 0
			if outer == nil {
				outer = clone
			}
			if container != nil {
				container.Children = append(container.Children, clone)
			}
			container = clone
			clones = append(clones, clone)
			if entry := p.activeFormattingEntryByName(template.Name); entry != nil && entry.current == nil {
				entry.current = clone
				p.formattingByNode[clone] = entry
			}
		}
		block.Parent = container
		container.Children = append(container.Children, block)
		// The outer clone is a virtual child of the formatting element's
		// original parent until the deferred split is materialized.  Record
		// that relationship before registering the clones as open so their
		// insertion containers are derived from the final chain.
		outer.Parent = formatting.Parent
		for _, clone := range clones {
			p.setFormattingOpen(clone, true)
		}
		splitNode = outer
	} else {
		block.Parent = formatting.Parent
	}
	p.adoptionSplit[formatting] = append(p.adoptionSplit[formatting], splitNode)
	p.adoptionDetached[block] = true
	p.recoverOpenElementsAtEOF = true
	return adoptionContinue, nil
}

func (p *HTMLParser) extendIncompleteAdoptionAtEOF(node *types.Node, tokenStart int) {
	if node == nil {
		return
	}
	if len(node.Children) > 0 {
		last := node.Children[len(node.Children)-1]
		if last.Type == types.ElementNode {
			p.extendIncompleteAdoptionAtEOF(last, tokenStart)
		} else if last.Type == types.TextNode && last.EndPos == tokenStart {
			last.EndPos = p.pos
			last.EndLine = p.line
			last.EndColumn = p.col
		}
	}
	if node.EndPos == 0 || node.EndPos == tokenStart {
		node.ContentEnd = p.pos
		node.EndPos = p.pos
		node.EndLine = p.line
		node.EndColumn = p.col
	}
	refreshNodeTextContent(node)
}

func (p *HTMLParser) cachedAdoptionFurthestBlock(formatting, current *types.Node) *types.Node {
	if p.adoptionUnwindPos == p.pos && p.adoptionUnwindFormatting == formatting {
		return p.adoptionUnwindBlock
	}
	block := adoptionFurthestBlock(formatting, current)
	p.adoptionUnwindPos = p.pos
	p.adoptionUnwindFormatting = formatting
	p.adoptionUnwindBlock = block
	return block
}

func (p *HTMLParser) cachedAdoptionInScope(formatting, current *types.Node) bool {
	if p.adoptionUnwindPos == p.pos && p.adoptionUnwindFormatting == formatting {
		return p.adoptionUnwindInScope
	}
	inScope := formattingInOrdinaryScope(formatting, current)
	p.adoptionUnwindPos = p.pos
	p.adoptionUnwindFormatting = formatting
	p.adoptionUnwindBlock = adoptionFurthestBlock(formatting, current)
	p.adoptionUnwindInScope = inScope
	return inScope
}

func adoptionGenericEndTarget(current *types.Node, name string) *types.Node {
	for node := current; node != nil; node = node.Parent {
		if node.Name == name {
			return node
		}
		if isHTMLSpecialElement(node.Name) {
			return nil
		}
	}
	return nil
}

func (p *HTMLParser) cachedGenericEndTarget(current *types.Node, name string) *types.Node {
	if p.genericEndPos == p.pos && p.genericEndName == name {
		return p.genericEndTarget
	}
	var target *types.Node
	if isParagraphClosingAncestorEndTag(name) {
		target = nearestOpenElementByName(current, name)
		if target != nil && !formattingInOrdinaryScope(target, current) {
			target = nil
		}
	} else {
		target = adoptionGenericEndTarget(current, name)
	}
	p.genericEndPos = p.pos
	p.genericEndName = name
	p.genericEndTarget = target
	return target
}

func (p *HTMLParser) removeActiveFormattingEntry(entry *formattingEntry) {
	if entry == nil {
		return
	}
	p.setFormattingOpen(entry.current, false)
	p.removeFormattingFamilyEntry(entry)
	for index := len(p.activeFormatting) - 1; index >= 0; index-- {
		if p.activeFormatting[index] == entry {
			p.activeFormatting = append(p.activeFormatting[:index], p.activeFormatting[index+1:]...)
			p.finishFormattingDocumentEndRecovery()
			return
		}
	}
}

func (p *HTMLParser) adoptionInnerFormatting(formatting, block *types.Node, closeStart, closeLine, closeColumn int) []*types.Node {
	var insideOut []*types.Node
	distance := 0
	for node := block.Parent; node != nil && node != formatting; node = node.Parent {
		distance++
		entry := p.formattingByNode[node]
		if entry == nil {
			node.ContentEnd = closeStart
			node.EndPos = closeStart
			node.EndLine = closeLine
			node.EndColumn = closeColumn
			p.adoptionClosed[node] = true
			p.adoptionIgnoredEnds[node.Name]++
			p.setFormattingOpen(node, false)
			continue
		}
		if distance > 3 {
			p.refreshFormattingText(node)
			node.ContentEnd = closeStart
			node.EndPos = closeStart
			node.EndLine = closeLine
			node.EndColumn = closeColumn
			p.adoptionClosed[node] = true
			p.adoptionIgnoredEnds[node.Name]++
			p.removeActiveFormatting(node)
			continue
		}
		insideOut = append(insideOut, node)
		// The source recursive frame is removed from the open-elements stack by
		// the adoption inner loop. Its active-formatting entry survives with a
		// replacement identity, but the stale source frame must return when the
		// furthest block finishes instead of waiting for an end tag that will be
		// consumed by the reconstructed clone.
		p.setFormattingOpen(node, false)
		p.adoptionReturnAtSplit[node] = true
		end := node.ContentStart
		if end == 0 && len(node.Children) > 0 {
			end = node.Children[0].StartPos
		}
		node.ContentEnd = end
		node.EndPos = end
		if line := p.elementContentStartLine[node]; line != 0 {
			node.EndLine = line
			node.EndColumn = p.elementContentStartColumn[node]
		} else if len(node.Children) > 0 {
			node.EndLine = node.Children[0].StartLine
			node.EndColumn = node.Children[0].StartColumn
		}
		delete(p.formattingText, node)
		delete(p.formattingByNode, node)
		entry.current = nil
	}
	wrappers := make([]*types.Node, len(insideOut))
	for index := range insideOut {
		wrappers[len(insideOut)-1-index] = insideOut[index]
	}
	for node := block.Parent; node != nil && node != formatting; node = node.Parent {
		if node.Type != types.ElementNode || isFormattingElement(node.Name) {
			continue
		}
		node.ContentEnd = closeStart
		node.EndPos = closeStart
		node.EndLine = closeLine
		node.EndColumn = closeColumn
		p.adoptionClosed[node] = true
	}
	return wrappers
}

func adoptionSpecialDescendants(node *types.Node) []*types.Node {
	var descendants []*types.Node
	var visit func(*types.Node)
	visit = func(parent *types.Node) {
		for _, child := range parent.Children {
			if child.Type != types.ElementNode {
				continue
			}
			visit(child)
			if isHTMLSpecialElement(child.Name) {
				descendants = append(descendants, child)
			}
		}
	}
	visit(node)
	return descendants
}

func (p *HTMLParser) consumePendingAdoptionEnd(node *types.Node) (bool, error) {
	pending := p.adoptionPendingEnds[node]
	if len(pending) == 0 {
		return false, nil
	}
	name, closing, ok := p.peekTagName()
	if !ok || !closing {
		return false, nil
	}
	match := -1
	for index, candidate := range pending {
		if name == candidate.Name {
			match = index
			break
		}
	}
	if match < 0 {
		return false, nil
	}
	tokenStart := p.pos
	closed := pending[match]
	_, emitted, err := p.parseClosingTag()
	if err != nil {
		return true, err
	}
	if !emitted {
		p.extendPreviousTextRange(closed, tokenStart)
		return true, nil
	}
	closed.ContentEnd = tokenStart
	closed.EndPos = p.pos
	closed.EndLine = p.line
	closed.EndColumn = p.col
	refreshNodeTextContent(closed)
	p.adoptionPendingEnds[node] = pending[match+1:]
	return true, nil
}

func (p *HTMLParser) finishPendingAdoptionAtEOF(node *types.Node) {
	pending := p.adoptionPendingEnds[node]
	if len(pending) == 0 || p.pos < len(p.content) {
		return
	}
	for _, current := range pending {
		current.ContentEnd = p.pos
		current.EndPos = p.pos
		current.EndLine = p.line
		current.EndColumn = p.col
	}
	delete(p.adoptionPendingEnds, node)
	refreshTreeText(node)
}

func (p *HTMLParser) adoptionWrapSpecialContents(block, formatting *types.Node, wrappers []*types.Node, closeEnd, closeLine, closeColumn, remaining int) {
	original := block.Children
	block.Children = nil
	segment := make([]*types.Node, 0, len(original))
	emitEmpty := len(original) == 0
	flush := func() {
		if len(segment) == 0 && !emitEmpty {
			return
		}
		emitEmpty = false
		formatClone := cloneFormattingNode(formatting, block)
		formatClone.StartPos, formatClone.StartLine, formatClone.StartColumn = 0, 0, 0
		formatClone.ContentStart = 0
		formatClone.Children = segment
		var parent = formatClone
		for _, wrapper := range wrappers {
			clone := cloneFormattingNode(wrapper, parent)
			clone.ContentEnd = closeEnd
			clone.EndPos = closeEnd
			clone.EndLine = closeLine
			clone.EndColumn = closeColumn
			clone.Children = parent.Children
			for _, child := range clone.Children {
				child.Parent = clone
			}
			parent.Children = []*types.Node{clone}
			parent = clone
		}
		for _, child := range formatClone.Children {
			child.Parent = formatClone
		}
		refreshTreeText(formatClone)
		if len(wrappers) > 0 {
			formatClone.StartPos = formatting.StartPos
			formatClone.StartLine = formatting.StartLine
			formatClone.StartColumn = formatting.StartColumn
			formatClone.ContentStart = formatting.ContentStart
			formatClone.EndPos = closeEnd
			formatClone.EndLine = closeLine
			formatClone.EndColumn = closeColumn
		}
		block.Children = append(block.Children, formatClone)
		segment = nil
	}
	for _, child := range original {
		if remaining > 1 && child.Type == types.ElementNode && isHTMLSpecialElement(child.Name) {
			flush()
			p.adoptionWrapSpecialContents(child, formatting, nil, closeEnd, closeLine, closeColumn, remaining-1)
			child.Parent = block
			block.Children = append(block.Children, child)
			continue
		}
		segment = append(segment, child)
	}
	flush()
}

func refreshTreeText(node *types.Node) {
	for _, child := range node.Children {
		if child.Type == types.ElementNode {
			refreshTreeText(child)
		}
	}
	refreshNodeTextContent(node)
}

func detachNodeShallow(node *types.Node) {
	if node == nil || node.Parent == nil {
		return
	}
	parent := node.Parent
	for index, child := range parent.Children {
		if child == node {
			parent.Children = append(parent.Children[:index], parent.Children[index+1:]...)
			break
		}
	}
	node.Parent = nil
}

func formattingInOrdinaryScope(formatting, current *types.Node) bool {
	for node := current; node != nil; node = node.Parent {
		if node == formatting {
			return true
		}
		if isOrdinaryScopeBoundary(node.Name) {
			return false
		}
	}
	return false
}

func adoptionFurthestBlock(formatting, current *types.Node) *types.Node {
	var block *types.Node
	for node := current; node != nil && node != formatting; node = node.Parent {
		if isHTMLSpecialElement(node.Name) {
			block = node
		}
	}
	return block
}

func (p *HTMLParser) appendDeferredAdoption(parent, child *types.Node) {
	siblings := p.adoptionSplit[child]
	if len(siblings) == 0 {
		return
	}
	delete(p.adoptionSplit, child)
	if p.closedForms[parent] {
		p.deferredFormSiblings[parent] = append(p.deferredFormSiblings[parent], siblings...)
		return
	}
	for _, sibling := range siblings {
		if sibling.Parent != nil && sibling.Parent.StartLine == 0 {
			if split := p.adoptionSplit[sibling.Parent]; len(split) > 0 {
				delete(p.adoptionSplit, sibling.Parent)
				for _, nested := range split {
					appendUniqueChild(parent, nested)
					p.appendNestedDeferredAdoption(parent, nested)
				}
				continue
			}
		}
		container := parent
		for _, entry := range p.activeFormatting[p.formattingMarkerStart:] {
			if entry.marker || entry.current == nil || entry.current == sibling || !p.openFormatting[entry.current] || p.formattingContainer[entry.current] != parent {
				continue
			}
			container = entry.current
		}
		appendUniqueChild(container, sibling)
		p.appendDeferredAdoption(parent, sibling)
		p.appendNestedDeferredAdoption(parent, sibling)
	}
}

func (p *HTMLParser) appendDeferredFormSiblings(parent, form *types.Node) {
	siblings := p.deferredFormSiblings[form]
	if len(siblings) == 0 {
		return
	}
	delete(p.deferredFormSiblings, form)
	delete(p.closedForms, form)
	for _, sibling := range siblings {
		if sibling.Parent != nil {
			detachNodeShallow(sibling)
		}
		appendUniqueChild(parent, sibling)
		p.appendNestedDeferredAdoption(parent, sibling)
	}
	refreshNodeTextContent(form)
}

func (p *HTMLParser) appendAllDeferredAdoption(parent *types.Node) {
	keys := make([]*types.Node, 0, len(p.adoptionSplit))
	for key := range p.adoptionSplit {
		keys = append(keys, key)
	}
	for _, key := range keys {
		siblings, present := p.adoptionSplit[key]
		if !present {
			continue
		}
		if len(siblings) == 0 {
			continue
		}
		delete(p.adoptionSplit, key)
		for _, sibling := range siblings {
			if sibling.Parent != nil {
				detachNodeShallow(sibling)
			}
			appendUniqueChild(parent, sibling)
			p.appendNestedDeferredAdoption(parent, sibling)
		}
	}
}

func appendUniqueChild(parent, child *types.Node) {
	if child.Parent != parent {
		child.Parent = parent
		parent.Children = append(parent.Children, child)
		return
	}
	for _, existing := range parent.Children {
		if existing == child {
			child.Parent = parent
			return
		}
	}
	child.Parent = parent
	parent.Children = append(parent.Children, child)
}

func (p *HTMLParser) detachDeferredAdoption(node *types.Node) {
	for owner, siblings := range p.adoptionSplit {
		for index, sibling := range siblings {
			if sibling == node {
				p.adoptionSplit[owner] = append(siblings[:index], siblings[index+1:]...)
				if len(p.adoptionSplit[owner]) == 0 {
					delete(p.adoptionSplit, owner)
				}
				return
			}
			// If the block lives inside an already queued synthetic wrapper,
			// keep that wrapper queued. Detaching the block below naturally
			// leaves the empty wrapper chain in the old output slot; the new
			// split for the adopted formatting element will emit the block
			// immediately afterwards.
			if nodeWithin(node, sibling) {
				return
			}
		}
	}
}

func (p *HTMLParser) appendNestedDeferredAdoption(parent, node *types.Node) {
	for _, child := range append([]*types.Node(nil), node.Children...) {
		if len(p.adoptionSplit[child]) > 0 {
			p.appendDeferredAdoption(parent, child)
			continue
		}
		p.appendNestedDeferredAdoption(parent, child)
	}
}

func isFormattingMarkerElement(name string) bool {
	switch name {
	case "applet", "object", "marquee", "template", "td", "th", "caption":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) addActiveFormatting(node *types.Node) {
	key := formattingFamilyKey(node)
	family := p.formattingFamilies[key]
	if len(family) >= 3 {
		evicted := family[0]
		family = family[1:]
		for index := len(p.activeFormatting) - 1; index >= 0; index-- {
			if p.activeFormatting[index] == evicted {
				p.activeFormatting = append(p.activeFormatting[:index], p.activeFormatting[index+1:]...)
				break
			}
		}
		p.removeFormattingNameAndNodeEntry(evicted)
	}
	entry := &formattingEntry{template: node, current: node, family: key, index: len(p.activeFormatting)}
	p.activeFormatting = append(p.activeFormatting, entry)
	p.formattingFamilies[key] = append(family, entry)
	p.formattingByName[node.Name] = append(p.formattingByName[node.Name], entry)
	p.formattingByNode[node] = entry
	p.formattingCounts[key] = len(p.formattingFamilies[key])
}

func formattingFamilyKey(node *types.Node) string {
	if node == nil {
		return ""
	}
	if len(node.Attributes) == 0 {
		return node.Name
	}
	if len(node.Attributes) == 1 {
		for name, value := range node.Attributes {
			return node.Name + "\x00" + name + "=" + value
		}
	}
	names := make([]string, 0, len(node.Attributes))
	for name := range node.Attributes {
		names = append(names, name)
	}
	sort.Strings(names)
	var key strings.Builder
	key.WriteString(node.Name)
	for _, name := range names {
		key.WriteByte(0)
		key.WriteString(name)
		key.WriteByte('=')
		key.WriteString(node.Attributes[name])
	}
	return key.String()
}

func (p *HTMLParser) clearActiveFormattingToMarker() {
	for len(p.activeFormatting) > 0 {
		last := len(p.activeFormatting) - 1
		marker := p.activeFormatting[last].marker
		if !marker {
			p.removeFormattingFamilyEntry(p.activeFormatting[last])
		}
		p.activeFormatting = p.activeFormatting[:last]
		if marker {
			p.finishFormattingDocumentEndRecovery()
			return
		}
	}
	p.finishFormattingDocumentEndRecovery()
}

func (p *HTMLParser) removeActiveFormatting(node *types.Node) {
	entry := p.formattingByNode[node]
	if entry == nil {
		return
	}
	p.setFormattingOpen(entry.current, false)
	p.removeFormattingFamilyEntry(entry)
	if last := len(p.activeFormatting) - 1; last >= 0 && p.activeFormatting[last] == entry {
		p.activeFormatting = p.activeFormatting[:last]
		p.finishFormattingDocumentEndRecovery()
		return
	}
	for index := len(p.activeFormatting) - 1; index >= 0; index-- {
		if p.activeFormatting[index] == entry {
			p.activeFormatting = append(p.activeFormatting[:index], p.activeFormatting[index+1:]...)
			p.finishFormattingDocumentEndRecovery()
			return
		}
	}
}

func (p *HTMLParser) finishFormattingDocumentEndRecovery() {
	if !p.formattingDocumentEnd || len(p.activeFormatting) > 0 {
		return
	}
	p.formattingDocumentEnd = false
	p.recoverOpenElementsAtEOF = p.formattingDocumentEndPrior
	p.formattingDocumentEndPrior = false
	if p.hasActiveListItem() || p.hasActiveSelectTree() || p.adoptionEOFBoundary != nil {
		p.recoverOpenElementsAtEOF = true
	}
}

func (p *HTMLParser) removeFormattingFamilyEntry(entry *formattingEntry) {
	if entry == nil || entry.family == "" {
		return
	}
	family := p.formattingFamilies[entry.family]
	for index, candidate := range family {
		if candidate == entry {
			family = append(family[:index], family[index+1:]...)
			break
		}
	}
	if len(family) == 0 {
		delete(p.formattingFamilies, entry.family)
		delete(p.formattingCounts, entry.family)
	} else {
		p.formattingFamilies[entry.family] = family
		p.formattingCounts[entry.family] = len(family)
	}
	p.removeFormattingNameAndNodeEntry(entry)
}

func (p *HTMLParser) removeFormattingNameAndNodeEntry(entry *formattingEntry) {
	name := entry.template.Name
	byName := p.formattingByName[name]
	if last := len(byName) - 1; last >= 0 && byName[last] == entry {
		byName = byName[:last]
	} else {
		for index, candidate := range byName {
			if candidate == entry {
				byName = append(byName[:index], byName[index+1:]...)
				break
			}
		}
	}
	if len(byName) == 0 {
		delete(p.formattingByName, name)
	} else {
		p.formattingByName[name] = byName
	}
	delete(p.formattingByNode, entry.template)
	if entry.current != nil {
		delete(p.formattingByNode, entry.current)
	}
}

func cloneFormattingNode(template, parent *types.Node) *types.Node {
	attributes := make(map[string]string, len(template.Attributes))
	for name, value := range template.Attributes {
		attributes[name] = value
	}
	attributeNamespaces := cloneStringMap(template.AttributeNamespaces)
	attributeLocalNames := cloneStringMap(template.AttributeLocalNames)
	attributePrefixes := cloneStringMap(template.AttributePrefixes)
	return &types.Node{
		Type: types.ElementNode, Name: template.Name, NamespaceURI: template.NamespaceURI, Attributes: attributes,
		AttributeNamespaces: attributeNamespaces, AttributeLocalNames: attributeLocalNames, AttributePrefixes: attributePrefixes,
		AttributeOrder: append([]string(nil), template.AttributeOrder...), Children: []*types.Node{}, Parent: parent,
		StartPos: template.StartPos, StartLine: template.StartLine, StartColumn: template.StartColumn,
		ContentStart: template.ContentStart,
	}
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func nodeWithin(node, ancestor *types.Node) bool {
	for current := node; current != nil; current = current.Parent {
		if current == ancestor {
			return true
		}
	}
	return false
}

func (p *HTMLParser) formattingInsertionParent(container *types.Node, reconstruct bool) *types.Node {
	if p.documentBody != nil && (container == p.documentHTML || container == p.documentRoot) && (reconstruct || p.hasOpenFormatting(p.documentBody)) {
		container = p.documentBody
	}
	if !reconstruct {
		if current := p.lastOpenFormattingNode(); current != nil && p.formattingContainer[current] == container {
			return current
		}
		return container
	}
	parent := container
	start := p.formattingMarkerStart
	lastOpen := start - 1
	if openNode := p.lastOpenFormattingNode(); openNode != nil {
		if p.formattingContainer[openNode] == container {
			parent = openNode
		}
		if entry := p.formattingByNode[openNode]; entry != nil {
			if entry.index >= start && entry.index < len(p.activeFormatting) && p.activeFormatting[entry.index] == entry {
				lastOpen = entry.index
			} else {
				for index := len(p.activeFormatting) - 1; index >= start; index-- {
					if p.activeFormatting[index] == entry {
						entry.index = index
						lastOpen = index
						break
					}
				}
			}
		}
	}
	for index := lastOpen + 1; index < len(p.activeFormatting); index++ {
		entry := p.activeFormatting[index]
		clone := cloneFormattingNode(entry.template, parent)
		parent.Children = append(parent.Children, clone)
		if entry.current != nil {
			delete(p.formattingByNode, entry.current)
		}
		entry.current = clone
		p.formattingByNode[clone] = entry
		p.setFormattingOpen(clone, true)
		parent = clone
	}
	return parent
}

func (p *HTMLParser) nextTokenReconstructsFormatting() bool {
	if p.pos >= len(p.content) || p.nextTokenIsComment() || p.nextTokenIsIgnoredDoctype() {
		return false
	}
	if p.peek() != '<' || !p.startsMarkupToken() {
		preview := *p
		node, _ := preview.parseTextNode(nil, preview.pos, preview.line, preview.col)
		return node != nil
	}
	name, closing, ok := p.peekTagName()
	if !ok || !p.currentTagTokenWillEmit() {
		return false
	}
	if closing {
		return name == "br"
	}
	// Batch 14A reconstructs on ordinary/generic and formatting starts. The
	// special block starts that only change the open-element stack wait for a
	// later character token, matching the in-body reconstruction step.
	if isFormattingElement(name) || !isHTMLSpecialElement(name) {
		return true
	}
	switch name {
	case "applet", "object", "marquee", "button", "area", "br", "embed", "img", "image", "input", "keygen", "wbr", "xmp", "select", "option", "optgroup", "math", "svg":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) hasOpenFormatting(container *types.Node) bool {
	current := p.lastOpenFormattingNode()
	return current != nil && p.formattingContainer[current] == container
}

func (p *HTMLParser) setFormattingOpen(node *types.Node, open bool) {
	if node == nil {
		return
	}
	if open {
		if !p.openFormatting[node] {
			p.openFormatting[node] = true
			if parent := node.Parent; parent != nil && p.openFormatting[parent] {
				p.formattingContainer[node] = p.formattingContainer[parent]
			} else {
				p.formattingContainer[node] = node.Parent
			}
			p.formattingOpenStack = append(p.formattingOpenStack, node)
			p.formattingTable[node] = p.activeTable
		}
		return
	}
	delete(p.openFormatting, node)
	delete(p.formattingContainer, node)
	delete(p.formattingTable, node)
}

func (p *HTMLParser) lastOpenFormattingNode() *types.Node {
	for len(p.formattingOpenStack) > 0 {
		last := len(p.formattingOpenStack) - 1
		node := p.formattingOpenStack[last]
		if p.openFormatting[node] {
			return node
		}
		p.formattingOpenStack = p.formattingOpenStack[:last]
	}
	return nil
}

func (p *HTMLParser) refreshOpenFormattingText() {
	if node := p.lastOpenFormattingNode(); node != nil {
		p.refreshFormattingChain(node)
	}
}

func (p *HTMLParser) refreshFormattingChain(parent *types.Node) {
	path := make([]*types.Node, 0, 8)
	for node := parent; node != nil; node = node.Parent {
		if !p.openFormatting[node] {
			break
		}
		path = append(path, node)
	}
	for _, node := range path {
		p.refreshFormattingText(node)
	}
}

func (p *HTMLParser) refreshFormattingText(node *types.Node) {
	if builder := p.formattingText[node]; builder != nil {
		node.TextContent = builder.String()
		delete(p.formattingText, node)
	} else {
		refreshNodeTextContent(node)
	}
	if len(node.Children) > 0 {
		last := node.Children[len(node.Children)-1]
		if node.StartLine != 0 && last.EndPos > node.EndPos {
			node.ContentEnd = last.EndPos
			node.EndPos = last.EndPos
			node.EndLine = last.EndLine
			node.EndColumn = last.EndColumn
		}
	}
}

func (p *HTMLParser) appendFormattingChild(container, parent, child *types.Node) {
	if parent == container {
		return
	}
	if child.Type == types.TextNode && len(parent.Children) > 0 && parent.Children[len(parent.Children)-1].Type == types.TextNode {
		mergeTextNodes(parent.Children[len(parent.Children)-1], child)
	} else {
		parent.Children = append(parent.Children, child)
	}
	value := child.TextContent
	switch child.Type {
	case types.TextNode:
		value = child.Value
	case types.CommentNode, types.ProcessingInstructionNode:
		value = ""
	}
	if value != "" {
		builder := p.formattingText[parent]
		if builder == nil {
			builder = &strings.Builder{}
			builder.WriteString(parent.TextContent)
			p.formattingText[parent] = builder
		}
		builder.WriteString(value)
	}
	if parent.StartLine != 0 {
		parent.ContentEnd = child.EndPos
		parent.EndPos = child.EndPos
		parent.EndLine = child.EndLine
		parent.EndColumn = child.EndColumn
	}
}

func (p *HTMLParser) finishFormattingWithin(container *types.Node, end, line, column int) {
	start := p.formattingOpenIndexWithin(container)
	for index := len(p.formattingOpenStack) - 1; index >= start; index-- {
		current := p.formattingOpenStack[index]
		if current == container || !p.openFormatting[current] {
			continue
		}
		p.refreshFormattingText(current)
		current.ContentEnd = end
		current.EndPos = end
		current.EndLine = line
		current.EndColumn = column
	}
}

func (p *HTMLParser) closeFormattingOnElementReturn(container *types.Node) {
	end := container.ContentEnd
	if end == 0 {
		end = container.EndPos
	}
	start := p.formattingOpenIndexWithin(container)
	for index := len(p.formattingOpenStack) - 1; index >= start; index-- {
		current := p.formattingOpenStack[index]
		if !p.openFormatting[current] {
			continue
		}
		entry := p.formattingByNode[current]
		if entry == nil {
			continue
		}
		// Adoption may leave a locationless formatting clone logically open
		// around the special element that is returning.  The clone must keep
		// owning later siblings until its own formatting end tag is handled.
		if current.StartLine == 0 && current != container && nodeWithin(container, current) {
			continue
		}
		if current != container {
			p.refreshFormattingText(current)
			if current.EndPos != 0 {
				entry.current = nil
				p.setFormattingOpen(current, false)
				continue
			}
			current.ContentEnd = end
			current.EndPos = end
			current.EndLine = container.EndLine
			current.EndColumn = container.EndColumn
		}
		entry.current = nil
		p.setFormattingOpen(current, false)
	}
}

func (p *HTMLParser) formattingOpenIndexWithin(container *types.Node) int {
	if p.lastOpenFormattingNode() == nil {
		return len(p.formattingOpenStack)
	}

	// Logically open formatting elements form a suffix of the open-formatting
	// stack for their nearest non-formatting container. Walk that live suffix
	// backwards and stop at the first open entry owned outside the returning
	// container. This avoids both a whole-stack scan and a walk through a deeply
	// reconstructed formatting DOM chain on every table row/cell return.
	start := len(p.formattingOpenStack)
	for index := len(p.formattingOpenStack) - 1; index >= 0; index-- {
		current := p.formattingOpenStack[index]
		if !p.openFormatting[current] {
			continue
		}
		base := p.formattingContainer[current]
		ownerTable := p.formattingTable[current]
		within := current == container || base == container
		if !within && ownerTable != nil && container == ownerTable {
			within = true
		} else if !within && ownerTable != nil && nodeWithin(container, ownerTable) {
			// Both nodes belong to the same table construction scope. Stop at
			// that table rather than following fostered/reconstructed ancestors
			// into arbitrarily deep prior document content.
			within = nodeWithinBefore(base, container, ownerTable)
		} else if !within {
			within = nodeWithin(base, container)
		}
		if !within {
			break
		}
		start = index
	}
	return start
}

func nodeWithinBefore(node, ancestor, boundary *types.Node) bool {
	for current := node; current != nil; current = current.Parent {
		if current == ancestor {
			return true
		}
		if current == boundary {
			return false
		}
	}
	return false
}

func (p *HTMLParser) formattingEntryForNode(node *types.Node) *formattingEntry {
	return p.formattingByNode[node]
}

func (p *HTMLParser) activeFormattingEntryByName(name string) *formattingEntry {
	entries := p.formattingByName[name]
	if len(entries) > 0 {
		return entries[len(entries)-1]
	}
	return nil
}

func isListStartScanStopper(name string) bool {
	// Current Chromium and parse5 continue the list-item scan through search,
	// despite the Living Standard's newer special-element classification.
	if name == "address" || name == "div" || name == "p" || name == "search" {
		return false
	}
	return isHTMLSpecialElement(name)
}

func isOrdinaryScopeBoundary(name string) bool {
	switch name {
	case "applet", "caption", "html", "marquee", "object", "select", "table", "td", "th", "template":
		return true
	default:
		return false
	}
}

func isListItemScopeBoundary(name string) bool {
	return isOrdinaryScopeBoundary(name) || name == "ol" || name == "ul"
}

func isHTMLSpecialElement(name string) bool {
	switch name {
	case "address", "applet", "area", "article", "aside", "base", "basefont", "bgsound", "blockquote", "body", "br", "button", "caption", "center", "col", "colgroup", "dd", "details", "dir", "div", "dl", "dt", "embed", "fieldset", "figcaption", "figure", "footer", "form", "frame", "frameset", "h1", "h2", "h3", "h4", "h5", "h6", "head", "header", "hgroup", "hr", "html", "iframe", "img", "input", "li", "link", "listing", "main", "marquee", "menu", "meta", "nav", "noembed", "noframes", "noscript", "object", "ol", "p", "param", "plaintext", "pre", "script", "search", "section", "select", "source", "style", "summary", "table", "tbody", "td", "template", "textarea", "tfoot", "th", "thead", "title", "tr", "track", "ul", "wbr", "xmp":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) hasActiveListItem() bool {
	return p.activeLIEnd != nil || p.activeDDEnd != nil || p.activeDTEnd != nil
}

func (p *HTMLParser) currentListDocumentEndTag() bool {
	name, closing, ok := p.peekTagName()
	// Body/html end tags change the document insertion mode without popping
	// still-open list items or active formatting elements.  Keep those virtual
	// open stacks alive so later after-body text can be reprocessed in body and
	// EOF can finish their browser ranges.
	activeDocumentEnd := p.listDocumentEndMode != "" || p.hasActiveListItem() || len(p.activeFormatting) > 0 ||
		(name == "html" && p.documentBodyEndPos > 0)
	return activeDocumentEnd && ok && closing && (name == "body" || name == "html") && p.currentTagTokenWillEmit()
}

func (p *HTMLParser) currentTokenReentersListBody() bool {
	if p.pos >= len(p.content) || p.nextTokenIsComment() {
		return false
	}
	remaining := p.content[p.pos:]
	if len(remaining) >= len("<!DOCTYPE") && strings.EqualFold(remaining[:len("<!DOCTYPE")], "<!DOCTYPE") {
		return false
	}
	if p.peek() == '<' && p.startsMarkupToken() {
		return true
	}
	preview := *p
	previewParent := &types.Node{Type: types.ElementNode, Name: "#preview", Children: []*types.Node{}}
	text, err := preview.parseTextNode(previewParent, preview.pos, preview.line, preview.col)
	if err != nil || text == nil {
		return true
	}
	return !isHTMLWhitespaceString(text.Value)
}

func (p *HTMLParser) nextTokenIsIgnoredDoctype() bool {
	remaining := p.content[p.pos:]
	return len(remaining) >= len("<!DOCTYPE") && strings.EqualFold(remaining[:len("<!DOCTYPE")], "<!DOCTYPE")
}

func (p *HTMLParser) nextTokenIsHTMLStart() bool {
	name, closing, ok := p.peekTagName()
	return ok && !closing && name == "html" && p.currentTagTokenWillEmit()
}

func (p *HTMLParser) mergeDuplicateHTMLStart() error {
	return p.mergeDuplicateStartInto(p.documentHTML, "html")
}

func (p *HTMLParser) mergeDuplicateStartInto(target *types.Node, expected string) error {
	if p.peek() != '<' {
		return nil
	}
	p.advance()
	name := p.parseTagName()
	if name != expected {
		return fmt.Errorf("expected %s start tag at position %d", expected, p.pos)
	}
	node := &types.Node{Type: types.ElementNode, Name: name, Attributes: make(map[string]string), AttributeOrder: []string{}}
	_, emitted := p.parseStartTagTail(node)
	if !emitted {
		return nil
	}
	if target == nil {
		return nil
	}
	if target.AttributeNamespaces == nil {
		target.AttributeNamespaces = make(map[string]string)
	}
	if target.AttributeLocalNames == nil {
		target.AttributeLocalNames = make(map[string]string)
	}
	if target.AttributePrefixes == nil {
		target.AttributePrefixes = make(map[string]string)
	}
	for _, name := range node.AttributeOrder {
		if _, exists := target.Attributes[name]; exists {
			continue
		}
		target.Attributes[name] = node.Attributes[name]
		target.AttributeOrder = append(target.AttributeOrder, name)
		target.AttributeNamespaces[name] = ""
		target.AttributeLocalNames[name] = name
		target.AttributePrefixes[name] = ""
	}
	return nil
}

func isHeadingName(name string) bool {
	return len(name) == 2 && name[0] == 'h' && name[1] >= '1' && name[1] <= '6'
}

func nodeIsDescendantOrSelf(node, ancestor *types.Node) bool {
	for current := node; current != nil; current = current.Parent {
		if current == ancestor {
			return true
		}
	}
	return false
}

func nodeHasAncestorName(node *types.Node, name string) bool {
	for current := node.Parent; current != nil; current = current.Parent {
		if current.Name == name {
			return true
		}
	}
	return false
}

func nearestOpenElementByName(node *types.Node, name string) *types.Node {
	for current := node; current != nil; current = current.Parent {
		if current.Name == name {
			return current
		}
	}
	return nil
}

func (p *HTMLParser) listItemUnwindTarget(current *types.Node) (*types.Node, bool) {
	if p.listUnwindCachePos == p.pos && p.listUnwindCacheTarget != nil {
		if current == p.listUnwindCacheTarget {
			name, closing, ok := p.peekTagName()
			if ok && closing && name == current.Name && current != p.activeLIEnd && current != p.activeDDEnd && current != p.activeDTEnd {
				p.clearListUnwindCache()
				return nil, false
			}
		}
		return p.listUnwindCacheTarget, p.listUnwindCacheExplicit
	}
	name, closing, ok := p.peekTagName()
	if !ok || !p.currentTagTokenWillEmit() {
		return nil, false
	}
	if p.activeLIStart == nil && p.activeDefinitionStart == nil && p.activeLIEnd == nil && p.activeDDEnd == nil && p.activeDTEnd == nil {
		return nil, false
	}
	var target *types.Node
	if !closing {
		switch name {
		case "li":
			target = p.activeLIStart
		case "dd", "dt":
			target = p.activeDefinitionStart
		default:
			return nil, false
		}
	} else {
		switch name {
		case "li":
			target = p.activeLIEnd
		case "dd":
			target = p.activeDDEnd
		case "dt":
			target = p.activeDTEnd
		default:
			if name == "body" || name == "html" {
				return nil, false
			}
			nearest := nearestOpenElementByName(current, name)
			if nearest == current {
				return nil, false
			}
			if nearest != nil {
				for _, candidate := range []*types.Node{p.activeLIEnd, p.activeDDEnd, p.activeDTEnd} {
					if candidate != nil && nodeIsDescendantOrSelf(nearest, candidate) {
						return p.cacheListUnwind(nearest, false)
					}
				}
			}
			if isHeadingName(name) {
				for currentNode := current; currentNode != nil; currentNode = currentNode.Parent {
					if !isHeadingName(currentNode.Name) {
						continue
					}
					for _, candidate := range []*types.Node{p.activeLIEnd, p.activeDDEnd, p.activeDTEnd} {
						if candidate != nil && nodeIsDescendantOrSelf(candidate, currentNode) {
							return p.cacheListUnwind(candidate, false)
						}
					}
					break
				}
			}
			if !isParagraphClosingAncestorEndTag(name) {
				return nil, false
			}
			if nearest != nil {
				for _, candidate := range []*types.Node{p.activeLIEnd, p.activeDDEnd, p.activeDTEnd} {
					if candidate == nil || !nodeIsDescendantOrSelf(current, candidate) {
						continue
					}
					if nodeIsDescendantOrSelf(candidate, nearest) && nearest != candidate {
						return p.cacheListUnwind(candidate, false)
					}
				}
			}
			for _, candidate := range []*types.Node{p.activeLIEnd, p.activeDDEnd, p.activeDTEnd} {
				if candidate != nil && nearestOpenElementByName(candidate.Parent, name) != nil && nodeIsDescendantOrSelf(current, candidate) {
					target = candidate
					break
				}
			}
			return p.cacheListUnwind(target, false)
		}
	}
	if target == nil {
		return nil, false
	}
	return p.cacheListUnwind(target, closing)
}

func (p *HTMLParser) cacheListUnwind(target *types.Node, explicit bool) (*types.Node, bool) {
	if target == nil {
		return nil, false
	}
	p.listUnwindCachePos = p.pos
	p.listUnwindCacheTarget = target
	p.listUnwindCacheExplicit = explicit
	return target, explicit
}

func (p *HTMLParser) clearListUnwindCache() {
	p.listUnwindCachePos = -1
	p.listUnwindCacheTarget = nil
	p.listUnwindCacheExplicit = false
}

func (p *HTMLParser) selectTreeUnwindTarget(current *types.Node) (*types.Node, string) {
	if p.selectUnwindPos == p.pos && p.selectUnwindTarget != nil {
		return p.selectUnwindTarget, p.selectUnwindAction
	}
	name, closing, ok := p.peekTagName()
	if !ok || !p.currentTagTokenWillEmit() {
		return nil, ""
	}
	var target *types.Node
	action := "implicit"
	if closing {
		action = "end"
		switch name {
		case "select":
			target = p.activeSelect
		case "option":
			target = p.activeOptionEnd
		case "optgroup":
			target = p.activeOptgroupEnd
		default:
			if p.activeSelect != nil {
				if name == "li" {
					target = p.activeLIEnd
				} else if name == "dd" {
					target = p.activeDDEnd
				} else if name == "dt" {
					target = p.activeDTEnd
				} else if isHeadingName(name) {
					target = nearestHeadingWithinBoundary(current, p.activeSelect)
					if target != nil && target.Name != name {
						action = "implicit"
					}
				} else {
					target = p.activeSelectNames[name]
					if isParagraphClosingAncestorEndTag(name) {
						target = p.activeSelectAllNames[name]
					}
				}
				if target == current {
					return nil, ""
				}
			}
		}
	} else {
		switch name {
		case "option":
			if p.activeSelect != nil {
				target = p.activeSelectOptionImplied
				if target == nil {
					target = p.activeOptionStart
				}
			} else if p.activeOptionStart == current {
				target = p.activeOptionStart
			}
		case "optgroup":
			if p.activeSelect != nil {
				target = p.activeSelectImpliedEnd
				if target == nil {
					target = p.activeOptionStart
				}
			} else if p.activeOptionStart == current {
				target = p.activeOptionStart
			}
			if target == nil && p.activeSelect != nil {
				target = p.activeOptgroupStart
			}
		case "select":
			target = p.activeSelect
			action = "nested-select"
		case "input":
			target = p.activeSelect
		case "hr":
			if p.activeSelect != nil {
				target = p.activeSelectImpliedEnd
				if target == nil {
					target = p.activeOptionStart
				}
				if target == nil {
					target = p.activeOptgroupStart
				}
			}
		}
	}
	if target == nil {
		return nil, ""
	}
	p.selectUnwindPos = p.pos
	p.selectUnwindTarget = target
	p.selectUnwindAction = action
	return target, action
}

func (p *HTMLParser) clearSelectUnwindCache() {
	p.selectUnwindPos = -1
	p.selectUnwindTarget = nil
	p.selectUnwindAction = ""
}

func (p *HTMLParser) preserveIgnoredTextRange(parent *types.Node, start, line, column int) {
	if len(parent.Children) > 0 {
		last := parent.Children[len(parent.Children)-1]
		if last.Type == types.TextNode {
			last.EndPos = p.pos
			last.EndLine = p.line
			last.EndColumn = p.col
			return
		}
	}
}

func (p *HTMLParser) currentIgnoredSelectEndTag(current *types.Node) bool {
	name, closing, ok := p.peekTagName()
	if !ok || !closing || !p.currentTagTokenWillEmit() {
		return false
	}
	if p.isSelfClosingTag(name) {
		return false
	}
	switch name {
	case "select":
		return p.activeSelect == nil
	case "option":
		return p.activeOptionEnd == nil
	case "optgroup":
		return p.activeOptgroupEnd == nil
	default:
		// p has its own in-body orphan-end recovery, including creation of a
		// locationless synthetic paragraph inside the select.
		if name == "p" || p.activeSelect == nil {
			return false
		}
		if isHeadingName(name) {
			return nearestHeadingWithinBoundary(current, p.activeSelect) == nil
		}
		if name == "li" {
			return p.activeLIEnd == nil
		}
		if name == "dd" {
			return p.activeDDEnd == nil
		}
		if name == "dt" {
			return p.activeDTEnd == nil
		}
		if isParagraphClosingAncestorEndTag(name) {
			return p.activeSelectAllNames[name] == nil
		}
		return p.activeSelectNames[name] == nil
	}
}

func nearestHeadingWithinBoundary(current, boundary *types.Node) *types.Node {
	for node := current; node != nil; node = node.Parent {
		if isHeadingName(node.Name) {
			return node
		}
		if node == boundary {
			break
		}
	}
	return nil
}

func (p *HTMLParser) consumeIgnoredStartTag() bool {
	if p.peek() != '<' {
		return false
	}
	p.advance()
	name := p.parseTagName()
	if name == "" {
		return false
	}
	temporary := &types.Node{Attributes: make(map[string]string)}
	_, emitted := p.parseStartTagTail(temporary)
	return emitted
}

func (p *HTMLParser) currentIgnoredListEndTag(current *types.Node) bool {
	name, closing, ok := p.peekTagName()
	if !ok || !closing || !p.currentTagTokenWillEmit() {
		return false
	}
	var target *types.Node
	switch name {
	case "li":
		target = p.activeLIEnd
	case "dd":
		target = p.activeDDEnd
	case "dt":
		target = p.activeDTEnd
	default:
		return false
	}
	return target == nil || !nodeIsDescendantOrSelf(current, target)
}

func (p *HTMLParser) listItemBlocksGenericAncestorEnd(current *types.Node, name string) bool {
	if isParagraphClosingAncestorEndTag(name) {
		return false
	}
	for _, target := range []*types.Node{p.activeLIEnd, p.activeDDEnd, p.activeDTEnd} {
		if target != nil && nodeIsDescendantOrSelf(current, target) && nodeHasAncestorName(target, name) {
			return true
		}
	}
	return false
}

func (p *HTMLParser) listItemIgnoresAbsentAncestorEnd(current *types.Node, name string) bool {
	if !isParagraphClosingAncestorEndTag(name) || !p.hasActiveListItem() {
		return false
	}
	if nearestOpenElementByName(current, name) != nil {
		return false
	}
	return true
}

func isParagraphButtonScopeBoundary(name string) bool {
	switch name {
	case "applet", "button", "caption", "html", "marquee", "object", "select", "table", "td", "th", "template":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) currentNestedButtonStart(current *types.Node) bool {
	if p.activeButton == nil || !nodeIsDescendantOrSelf(current, p.activeButton) {
		return false
	}
	name, closing, ok := p.peekTagName()
	return ok && !closing && name == "button" && p.currentTagTokenWillEmit()
}

func isParagraphAncestorScopeBoundary(name string) bool {
	switch name {
	case "applet", "caption", "html", "marquee", "object", "select", "table", "td", "th", "template":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) paragraphInButtonScope(current *types.Node) *types.Node {
	return p.activeParagraph
}

func isParagraphClosingStartTag(name string) bool {
	switch name {
	case "address", "article", "aside", "blockquote", "center", "details", "dialog", "dir", "div", "dl",
		"fieldset", "figcaption", "figure", "footer", "form", "header", "hgroup", "hr", "listing", "main",
		"menu", "nav", "ol", "p", "plaintext", "pre", "search", "section", "summary", "ul", "xmp",
		"dd", "dt", "li", "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) paragraphClosingStartTag(name string) bool {
	if name == "table" {
		return !p.quirksMode
	}
	return isParagraphClosingStartTag(name)
}

func isParagraphClosingAncestorEndTag(name string) bool {
	switch name {
	case "address", "applet", "article", "aside", "blockquote", "button", "center", "dd", "details", "dialog", "dir", "div", "dl", "dt",
		"fieldset", "figcaption", "figure", "footer", "form", "h1", "h2", "h3", "h4", "h5", "h6", "header", "hgroup", "html", "li", "listing", "main", "marquee", "menu",
		"nav", "object", "ol", "pre", "search", "section", "summary", "ul", "body":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) shouldUnwindForParagraph(current *types.Node) bool {
	paragraph := p.paragraphInButtonScope(current)
	if paragraph == nil {
		return false
	}
	if !p.currentTagTokenWillEmit() {
		return false
	}

	name, closing, ok := p.peekTagName()
	if !ok {
		return false
	}
	if !closing {
		return p.paragraphClosingStartTag(name)
	}
	if name == "p" {
		return current != paragraph
	}
	if !isParagraphClosingAncestorEndTag(name) {
		return false
	}
	_, closesAncestor := p.activeParagraphAncestors[name]
	return closesAncestor
}

func (p *HTMLParser) currentTagTokenWillEmit() (emits bool) {
	if p.pos >= len(p.content) || p.peek() != '<' {
		return false
	}
	p.peekTagName()
	if p.tagPreviewEmissionKnown {
		return p.tagPreviewEmits
	}
	defer func() {
		p.tagPreviewEmits = emits
		p.tagPreviewEmissionKnown = true
	}()
	position := p.pos + 1
	if position < len(p.content) && p.content[position] == '/' {
		position++
	}
	for position < len(p.content) && !isWhitespace(p.content[position]) && p.content[position] != '/' && p.content[position] != '>' {
		position++
	}

	state := beforeAttributeNameState
	for position < len(p.content) {
		c := p.content[position]
		switch state {
		case beforeAttributeNameState:
			switch {
			case isWhitespace(c):
				position++
			case c == '/':
				position++
				state = selfClosingStartTagState
			case c == '>':
				return true
			default:
				if c == '=' {
					position++
				}
				state = attributeNameState
			}
		case attributeNameState:
			switch {
			case isWhitespace(c):
				position++
				state = afterAttributeNameState
			case c == '/' || c == '>':
				state = afterAttributeNameState
			case c == '=':
				position++
				state = beforeAttributeValueState
			default:
				position++
			}
		case afterAttributeNameState:
			switch {
			case isWhitespace(c):
				position++
			case c == '/':
				position++
				state = selfClosingStartTagState
			case c == '=':
				position++
				state = beforeAttributeValueState
			case c == '>':
				return true
			default:
				state = attributeNameState
			}
		case beforeAttributeValueState:
			switch {
			case isWhitespace(c):
				position++
			case c == '"':
				position++
				state = attributeValueDoubleQuotedState
			case c == '\'':
				position++
				state = attributeValueSingleQuotedState
			case c == '>':
				return true
			default:
				state = attributeValueUnquotedState
			}
		case attributeValueDoubleQuotedState:
			position++
			if c == '"' {
				state = afterAttributeValueQuotedState
			}
		case attributeValueSingleQuotedState:
			position++
			if c == '\'' {
				state = afterAttributeValueQuotedState
			}
		case attributeValueUnquotedState:
			switch {
			case isWhitespace(c):
				position++
				state = beforeAttributeNameState
			case c == '>':
				return true
			default:
				position++
			}
		case afterAttributeValueQuotedState:
			switch {
			case isWhitespace(c):
				position++
				state = beforeAttributeNameState
			case c == '/':
				position++
				state = selfClosingStartTagState
			case c == '>':
				return true
			default:
				state = beforeAttributeNameState
			}
		case selfClosingStartTagState:
			if c == '>' {
				return true
			}
			state = beforeAttributeNameState
		}
	}
	return false
}

func (p *HTMLParser) parseTableElement(table *types.Node, contentStart int) (*types.Node, error) {
	previousTable := p.activeTable
	p.activeTable = table
	defer func() { p.activeTable = previousTable }()
	for p.pos < len(p.content) {
		if p.currentTemplateEndCloses(table) {
			p.finishTableImplicit(table, contentStart)
			return table, nil
		}
		name, closing, tag := p.peekTagName()
		if tag && !p.currentTagTokenWillEmit() {
			if _, err := p.consumeDiscardedTableToken(table); err != nil {
				return nil, err
			}
			continue
		}
		if tag && closing && name == "table" {
			if err := p.closeTableNode(table); err != nil {
				return nil, err
			}
			table.ContentStart = contentStart
			return table, nil
		}
		if tag && closing && (name == "body" || name == "html") {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if tag && !closing {
			switch name {
			case "template":
				child, err := p.parseNode(table)
				if err != nil {
					return nil, err
				}
				p.appendTableChild(table, child)
				continue
			case "table":
				p.finishTableImplicit(table, contentStart)
				return table, nil
			case "caption":
				caption, err := p.parseTableLooseContainer(table, "caption", map[string]bool{"caption": true, "col": true, "colgroup": true, "tbody": true, "tfoot": true, "thead": true, "tr": true})
				if err != nil {
					return nil, err
				}
				p.appendTableChild(table, caption)
				continue
			case "col":
				colgroup, err := p.parseTableColgroup(table, true)
				if err != nil {
					return nil, err
				}
				p.appendTableChild(table, colgroup)
				continue
			case "colgroup":
				colgroup, err := p.parseTableColgroup(table, false)
				if err != nil {
					return nil, err
				}
				p.appendTableChild(table, colgroup)
				continue
			case "tbody", "thead", "tfoot":
				section, err := p.parseTableSection(table, false)
				if err != nil {
					return nil, err
				}
				p.appendTableChild(table, section)
				continue
			case "tr", "td", "th":
				section, err := p.parseTableSection(table, true)
				if err != nil {
					return nil, err
				}
				p.appendTableChild(table, section)
				continue
			}
		}
		if tag && !closing && (name == "style" || name == "script") {
			child, err := p.parseNode(table)
			if err != nil {
				return nil, err
			}
			p.appendTableChild(table, child)
			continue
		}
		if tag && closing && isTableStructuralName(name) {
			_, _, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			continue
		}
		child, err := p.parseTableFosterNode(table)
		if err != nil {
			return nil, err
		}
		if child == nil {
			continue
		}
		if child.Type == types.CommentNode || child.Type == types.ProcessingInstructionNode || (child.Type == types.TextNode && isHTMLWhitespaceString(child.Value)) {
			p.appendTableChild(table, child)
		} else {
			p.fosterTableChild(table, child)
		}
	}
	p.recoverOpenElementsAtEOF = true
	p.finishTableImplicit(table, contentStart)
	return table, nil
}

func (p *HTMLParser) parseTableSection(table *types.Node, synthetic bool) (*types.Node, error) {
	var section *types.Node
	if synthetic {
		section = syntheticTableNode("tbody", table)
	} else {
		var err error
		section, err = p.parseTableOpenElement(table)
		if err != nil {
			return nil, err
		}
	}
	if err := p.parseRowsIntoSection(section); err != nil {
		return nil, err
	}
	return section, nil
}

func (p *HTMLParser) parseRowsIntoSection(section *types.Node) error {
	for p.pos < len(p.content) {
		if p.currentTemplateEndCloses(section) {
			p.finishTableImplicit(section, section.ContentStart)
			return nil
		}
		name, closing, tag := p.peekTagName()
		if tag && !p.currentTagTokenWillEmit() {
			if _, err := p.consumeDiscardedTableToken(section); err != nil {
				return err
			}
			continue
		}
		if tag {
			if closing {
				if name == section.Name {
					return p.closeTableNode(section)
				}
				if name == "table" || name == "tbody" || name == "thead" || name == "tfoot" {
					if name == "table" || name == section.Name {
						p.finishTableImplicit(section, section.ContentStart)
						return nil
					}
					_, _, err := p.parseClosingTag()
					if err != nil {
						return err
					}
					continue
				}
				if name == "body" || name == "caption" || name == "col" || name == "colgroup" || name == "html" || name == "tr" || name == "td" || name == "th" {
					_, _, err := p.parseClosingTag()
					if err != nil {
						return err
					}
					continue
				}
			} else {
				switch name {
				case "template":
					child, err := p.parseNode(section)
					if err != nil {
						return err
					}
					p.appendTableChild(section, child)
					continue
				case "tr":
					row, err := p.parseTableRow(section, false)
					if err != nil {
						return err
					}
					p.appendTableChild(section, row)
					continue
				case "td", "th":
					row, err := p.parseTableRow(section, true)
					if err != nil {
						return err
					}
					p.appendTableChild(section, row)
					continue
				case "tbody", "thead", "tfoot", "caption", "col", "colgroup", "table":
					p.finishTableImplicit(section, section.ContentStart)
					return nil
				}
			}
		}
		if consumed, err := p.consumeTemplatePseudoAbsentEnd(section); consumed || err != nil {
			if err != nil {
				return err
			}
			continue
		}
		if p.templatePseudoTableAnythingElse(section) {
			p.finishTableImplicit(section, section.ContentStart)
			return nil
		}
		child, err := p.parseTableFosterNodeInMode(section, section.Name, false)
		if err != nil {
			return err
		}
		if child != nil {
			if child.Type == types.CommentNode || child.Type == types.ProcessingInstructionNode || (child.Type == types.TextNode && isHTMLWhitespaceString(child.Value)) {
				p.appendTableChild(section, child)
			} else {
				p.fosterTableChild(p.activeTable, child)
			}
		}
	}
	p.recoverOpenElementsAtEOF = true
	p.finishTableImplicit(section, section.ContentStart)
	return nil
}

func (p *HTMLParser) parseTableRow(section *types.Node, synthetic bool) (*types.Node, error) {
	var row *types.Node
	if synthetic {
		row = syntheticTableNode("tr", section)
	} else {
		var err error
		row, err = p.parseTableOpenElement(section)
		if err != nil {
			return nil, err
		}
	}
	for p.pos < len(p.content) {
		if p.currentTemplateEndCloses(row) {
			p.finishTableImplicit(row, row.ContentStart)
			return row, nil
		}
		name, closing, tag := p.peekTagName()
		if tag && !p.currentTagTokenWillEmit() {
			if _, err := p.consumeDiscardedTableToken(row); err != nil {
				return nil, err
			}
			continue
		}
		if tag {
			if !closing && name == "template" {
				child, err := p.parseNode(row)
				if err != nil {
					return nil, err
				}
				p.appendTableChild(row, child)
				continue
			}
			if closing && name == "tr" {
				return row, p.closeTableNode(row)
			}
			if closing && (name == "body" || name == "caption" || name == "col" || name == "colgroup" || name == "html" || name == "td" || name == "th" || ((name == "tbody" || name == "thead" || name == "tfoot") && name != section.Name)) {
				_, _, err := p.parseClosingTag()
				if err != nil {
					return nil, err
				}
				continue
			}
			if (closing && (name == "table" || name == section.Name)) || (!closing && (name == "tr" || name == "tbody" || name == "thead" || name == "tfoot" || name == "caption" || name == "col" || name == "colgroup" || name == "table")) {
				p.finishTableImplicit(row, row.ContentStart)
				return row, nil
			}
			if !closing && (name == "td" || name == "th") {
				cell, err := p.parseTableCell(row)
				if err != nil {
					return nil, err
				}
				p.appendTableChild(row, cell)
				continue
			}
		}
		if consumed, err := p.consumeTemplatePseudoAbsentEnd(row); consumed || err != nil {
			if err != nil {
				return nil, err
			}
			continue
		}
		if p.templatePseudoTableAnythingElse(row) {
			p.finishTableImplicit(row, row.ContentStart)
			return row, nil
		}
		sectionName := ""
		if section != nil {
			sectionName = section.Name
		}
		child, err := p.parseTableFosterNodeInMode(row, sectionName, true)
		if err != nil {
			return nil, err
		}
		if child != nil {
			if child.Type == types.CommentNode || child.Type == types.ProcessingInstructionNode || (child.Type == types.TextNode && isHTMLWhitespaceString(child.Value)) {
				p.appendTableChild(row, child)
			} else {
				p.fosterTableChild(p.activeTable, child)
			}
		}
	}
	p.recoverOpenElementsAtEOF = true
	p.finishTableImplicit(row, row.ContentStart)
	return row, nil
}

func (p *HTMLParser) parseTableCell(row *types.Node) (*types.Node, error) {
	cell, err := p.parseTableOpenElement(row)
	if err != nil {
		return nil, err
	}
	previousFormattingCounts := p.formattingCounts
	previousFormattingFamilies := p.formattingFamilies
	previousFormattingByName := p.formattingByName
	previousFormattingMarkerStart := p.formattingMarkerStart
	p.activeFormatting = append(p.activeFormatting, &formattingEntry{marker: true})
	p.formattingMarkerStart = len(p.activeFormatting)
	p.formattingCounts = make(map[string]int)
	p.formattingFamilies = make(map[string][]*formattingEntry)
	p.formattingByName = make(map[string][]*formattingEntry)
	defer func() {
		p.clearActiveFormattingToMarker()
		p.formattingCounts = previousFormattingCounts
		p.formattingFamilies = previousFormattingFamilies
		p.formattingByName = previousFormattingByName
		p.formattingMarkerStart = previousFormattingMarkerStart
	}()
	defer p.closeFormattingOnElementReturn(cell)
	sectionName := ""
	if row.Parent != nil {
		sectionName = row.Parent.Name
	}
	for p.pos < len(p.content) {
		if p.currentTemplateEndCloses(cell) {
			p.finishTableImplicit(cell, cell.ContentStart)
			return cell, nil
		}
		name, closing, tag := p.peekTagName()
		if tag && !p.currentTagTokenWillEmit() {
			if _, err := p.consumeDiscardedTableToken(cell); err != nil {
				return nil, err
			}
			continue
		}
		if consumed, consumeErr := p.consumeForeignIgnoredEnd(cell); consumed || consumeErr != nil {
			if consumeErr != nil {
				return nil, consumeErr
			}
			continue
		}
		if handled, formErr := p.handleFormToken(cell, false); handled || formErr != nil {
			if formErr != nil {
				return nil, formErr
			}
			continue
		}
		if tag {
			if closing && name == cell.Name {
				return cell, p.closeTableNode(cell)
			}
			if closing && (name == "td" || name == "th") {
				_, _, ignoreErr := p.parseClosingTag()
				if ignoreErr != nil {
					return nil, ignoreErr
				}
				continue
			}
			if closing && (name == "tbody" || name == "thead" || name == "tfoot") && name != sectionName {
				_, _, ignoreErr := p.parseClosingTag()
				if ignoreErr != nil {
					return nil, ignoreErr
				}
				continue
			}
			if closing && (name == "body" || name == "caption" || name == "col" || name == "colgroup" || name == "html") {
				_, _, ignoreErr := p.parseClosingTag()
				if ignoreErr != nil {
					return nil, ignoreErr
				}
				continue
			}
			if closing && name == "tr" && (cell.Parent == nil || cell.Parent.Name != "tr") {
				if _, _, ignoreErr := p.parseClosingTag(); ignoreErr != nil {
					return nil, ignoreErr
				}
				continue
			}
			if closing && name == "table" && p.activeTable == nil {
				if _, _, ignoreErr := p.parseClosingTag(); ignoreErr != nil {
					return nil, ignoreErr
				}
				continue
			}
			if (closing && (name == "table" || name == sectionName || name == "tr")) || (!closing && (name == "td" || name == "th" || name == "tr" || name == "tbody" || name == "thead" || name == "tfoot" || name == "caption" || name == "col" || name == "colgroup")) {
				p.finishTableImplicit(cell, cell.ContentStart)
				return cell, nil
			}
		}
		if consumed, consumeErr := p.consumeTemplatePseudoAbsentEnd(cell); consumed || consumeErr != nil {
			if consumeErr != nil {
				return nil, consumeErr
			}
			continue
		}
		if name, closing, tag := p.peekTagName(); tag && closing && isCoreAdoptionSubject(name) && p.activeFormattingEntryByName(name) == nil {
			start := p.pos
			if _, _, closeErr := p.parseClosingTag(); closeErr != nil {
				return nil, closeErr
			}
			p.extendPreviousTextRange(cell, start)
			continue
		}
		if name, closing, tag := p.peekTagName(); tag && closing && isCoreAdoptionSubject(name) {
			action, adoptionErr := p.handleCoreAdoption(cell, name, cell.TextContent, cell.ContentStart, true)
			if adoptionErr != nil {
				return nil, adoptionErr
			}
			if action != adoptionNone {
				continue
			}
		}
		previousCell, previousSection := p.tableCellName, p.tableCellSection
		p.tableCellMode++
		p.tableCellName, p.tableCellSection = cell.Name, sectionName
		insertionParent := p.formattingInsertionParent(cell, p.nextTokenReconstructsFormatting())
		child, childErr := p.parseNode(insertionParent)
		p.tableCellMode--
		p.tableCellName, p.tableCellSection = previousCell, previousSection
		if childErr != nil {
			return nil, childErr
		}
		if child != nil {
			if insertionParent != cell {
				p.appendFormattingChild(cell, insertionParent, child)
				refreshNodeTextContent(cell)
			} else {
				p.appendTableChild(cell, child)
			}
			p.appendDeferredAdoption(cell, child)
		}
	}
	p.recoverOpenElementsAtEOF = true
	p.finishTableImplicit(cell, cell.ContentStart)
	return cell, nil
}

func (p *HTMLParser) parseTableOpenElement(parent *types.Node) (*types.Node, error) {
	start, line, column := p.pos, p.line, p.col
	if p.peek() != '<' {
		return nil, fmt.Errorf("expected table tag at position %d", p.pos)
	}
	p.advance()
	name := p.parseTagName()
	node := &types.Node{Type: types.ElementNode, Name: name, NamespaceURI: htmlElementNamespace, Attributes: make(map[string]string), AttributeOrder: []string{}, Children: []*types.Node{}, Parent: parent, StartPos: start, StartLine: line, StartColumn: column}
	if owner := p.templateOwner[parent]; owner != nil {
		p.templateOwner[node] = owner
	}
	_, emitted := p.parseStartTagTail(node)
	if !emitted {
		return nil, fmt.Errorf("incomplete table tag at position %d", start)
	}
	node.ContentStart = p.pos
	return node, nil
}

func isTableStructuralName(name string) bool {
	switch name {
	case "table", "caption", "colgroup", "col", "tbody", "tfoot", "thead", "tr", "td", "th":
		return true
	default:
		return false
	}
}

func isCaptionIgnoredEndTag(name string) bool {
	switch name {
	case "body", "col", "colgroup", "html", "tbody", "td", "tfoot", "th", "thead", "tr":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) currentFosterTransitionWillEmit() bool {
	name, closing, ok := p.peekTagName()
	if !ok || !p.currentTagTokenWillEmit() {
		return false
	}
	if !closing {
		switch name {
		case "table", "caption", "col", "colgroup", "tbody", "tfoot", "thead", "tr", "td", "th":
			return true
		}
	}
	if !closing {
		return false
	}
	if name == "table" {
		return true
	}
	if p.tableFosterRow && name == "tr" {
		return true
	}
	return p.tableFosterSection != "" && name == p.tableFosterSection
}

func (p *HTMLParser) currentFosterIgnoredEndTag() bool {
	name, closing, ok := p.peekTagName()
	if !ok || !closing || !p.currentTagTokenWillEmit() || name == "table" || (p.tableFosterRow && name == "tr") || (p.tableFosterSection != "" && name == p.tableFosterSection) {
		return false
	}
	return isTableStructuralName(name) || name == "body" || name == "html" || name == "noscript"
}

func (p *HTMLParser) currentCellTransitionWillEmit() bool {
	name, closing, ok := p.peekTagName()
	if !ok || !p.currentTagTokenWillEmit() {
		return false
	}
	if !closing {
		return isTableStructuralName(name) && name != "table"
	}
	return name == "table" || name == "tr" || name == p.tableCellName || (p.tableCellSection != "" && name == p.tableCellSection)
}

func (p *HTMLParser) currentCellIgnoredEndTag() bool {
	name, closing, ok := p.peekTagName()
	if !ok || !closing || !p.currentTagTokenWillEmit() || name == "table" || name == "tr" || name == p.tableCellName || (p.tableCellSection != "" && name == p.tableCellSection) {
		return false
	}
	return isTableStructuralName(name) || name == "body" || name == "html"
}

func (p *HTMLParser) currentCaptionTransitionWillEmit() bool {
	name, closing, ok := p.peekTagName()
	if !ok || !p.currentTagTokenWillEmit() {
		return false
	}
	if closing {
		return name == "caption" || name == "table"
	}
	switch name {
	case "caption", "col", "colgroup", "tbody", "tfoot", "thead", "tr", "td", "th":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) currentCaptionIgnoredEndTag() bool {
	name, closing, ok := p.peekTagName()
	return ok && closing && p.currentTagTokenWillEmit() && isCaptionIgnoredEndTag(name)
}

func (p *HTMLParser) consumeDiscardedTableToken(parent *types.Node) (*types.Node, error) {
	tokenStart := p.pos
	child, err := p.parseNode(parent)
	if err != nil {
		return nil, err
	}
	if child == nil {
		p.extendPreviousTextRange(parent, tokenStart)
		p.recoverOpenElementsAtEOF = true
	}
	return child, nil
}

func (p *HTMLParser) parseTableFosterNode(parent *types.Node) (*types.Node, error) {
	return p.parseTableFosterNodeInMode(parent, "", false)
}

// handleFormToken implements the whole-document form element pointer. The
// pointer and logical-open state are deliberately separate from DOM Parent:
// a non-current form end removes the pointed form from the open-elements stack
// while its still-open descendants remain DOM children of that form.
func (p *HTMLParser) handleFormToken(current *types.Node, inTable bool) (bool, error) {
	name, closing, tag := p.peekTagName()
	if !tag || name != "form" || !p.currentTagTokenWillEmit() {
		return false, nil
	}

	if !closing {
		if p.templateOwnerForNode(current) != nil {
			mode := templateModeInitial
			if len(p.templateInsertionModes) > 0 {
				mode = p.templateInsertionModes[len(p.templateInsertionModes)-1]
			}
			if inTable || mode != templateModeBody {
				p.consumeIgnoredStartTag()
				return true, nil
			}
			// The form pointer is ignored for an in-body form start while a
			// template is open. parseElement dynamically restores the surrounding
			// pointer when each template-local form frame returns.
			return false, nil
		}
		if p.activeForm != nil {
			p.consumeIgnoredStartTag()
			return true, nil
		}
		if inTable {
			form, err := p.parseTableOpenElement(current)
			if err != nil {
				return true, err
			}
			if form != nil {
				// The in-table rule inserts and immediately pops the source-backed
				// form. Keep the library's parse5-compatible start-only range.
				form.ContentStart = form.StartPos
				form.ContentEnd = form.StartPos
				form.EndPos = form.StartPos
				form.EndLine = form.StartLine
				form.EndColumn = form.StartColumn
				appendUniqueChild(current, form)
				p.activeForm = form
				p.openForms[form] = false
			}
			return true, nil
		}
		return false, nil
	}

	tokenStart := p.pos
	pointed := p.activeForm
	p.activeForm = nil // The algorithm clears the pointer before scope checks.
	if _, emitted, err := p.parseClosingTag(); err != nil {
		return true, err
	} else if !emitted {
		p.extendPreviousTextRange(current, tokenStart)
		return true, nil
	}
	if pointed == nil || !p.openForms[pointed] || !formattingInOrdinaryScope(pointed, current) {
		return true, nil
	}
	p.openForms[pointed] = false
	pointed.ContentEnd = tokenStart
	pointed.EndPos = p.pos
	pointed.EndLine = p.line
	pointed.EndColumn = p.col
	if pointed != current {
		p.closedForms[pointed] = true
		return true, nil
	}
	p.closedForms[pointed] = true
	return true, nil
}

func (p *HTMLParser) parseTableFosterNodeInMode(parent *types.Node, section string, row bool) (*types.Node, error) {
	previousSection, previousRow := p.tableFosterSection, p.tableFosterRow
	p.tableFosterMode++
	p.tableFosterSection, p.tableFosterRow = section, row
	defer func() {
		p.tableFosterMode--
		p.tableFosterSection, p.tableFosterRow = previousSection, previousRow
	}()
	effectiveCurrent := parent
	if open := p.lastOpenFormattingNode(); open != nil && p.formattingContainer[open] == parent {
		effectiveCurrent = open
	}
	// With scripting disabled, an in-table noscript start is fostered and its
	// frame closes when table structure is reprocessed. Its later source end
	// tag is therefore absent from table scope and is ignored.
	if name, closing, ok := p.peekTagName(); ok && closing && name == "noscript" && !p.scriptingEnabled && p.currentTagTokenWillEmit() {
		if _, _, err := p.parseClosingTag(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if handled, err := p.handleFormToken(effectiveCurrent, true); handled || err != nil {
		return nil, err
	}
	if hiddenInput, consumed, err := p.parseHiddenInputInTable(effectiveCurrent); err != nil {
		return nil, err
	} else if consumed {
		if hiddenInput != nil {
			if effectiveCurrent == parent {
				p.appendTableChild(parent, hiddenInput)
			} else {
				p.appendFormattingChild(parent, effectiveCurrent, hiddenInput)
			}
		}
		return nil, nil
	}
	if action, err := p.handleSpecialFormattingStart(effectiveCurrent, effectiveCurrent.TextContent, effectiveCurrent.ContentStart); err != nil {
		return nil, err
	} else if action != adoptionNone {
		return nil, nil
	}
	if name, closing, ok := p.peekTagName(); ok && closing && isCoreAdoptionSubject(name) {
		action, err := p.handleCoreAdoption(effectiveCurrent, name, effectiveCurrent.TextContent, effectiveCurrent.ContentStart, true)
		if err != nil {
			return nil, err
		}
		if action != adoptionNone {
			return nil, nil
		}
	}
	insertionParent := p.formattingInsertionParent(parent, p.nextTableTokenReconstructsFormatting(parent))
	if insertionParent == parent {
		return p.parseNode(parent)
	}
	outerIndex := len(parent.Children) - 1
	outer := parent.Children[outerIndex]
	parent.Children = parent.Children[:outerIndex]
	child, err := p.parseNode(insertionParent)
	if err != nil {
		return nil, err
	}
	if child != nil {
		p.appendFormattingChild(parent, insertionParent, child)
		outer.TextContent = child.TextContent
		outer.ContentEnd = child.EndPos
		outer.EndPos = child.EndPos
		outer.EndLine = child.EndLine
		outer.EndColumn = child.EndColumn
	}
	return outer, nil
}

// parseHiddenInputInTable implements the in-table hidden-input exception.
// It previews the fully tokenized attributes so character references and the
// tokenizer's first-duplicate-wins rule are honored, then consumes the real
// void element only when its type value is exactly ASCII-folded "hidden".
func (p *HTMLParser) parseHiddenInputInTable(parent *types.Node) (*types.Node, bool, error) {
	name, closing, tag := p.peekTagName()
	if !tag || closing || name != "input" || !p.currentTagTokenWillEmit() {
		return nil, false, nil
	}

	preview := *p
	preview.advance() // '<'
	if preview.parseTagName() != "input" {
		return nil, false, nil
	}
	attributes := &types.Node{Attributes: make(map[string]string), AttributeOrder: []string{}}
	if _, emitted := preview.parseStartTagTail(attributes); !emitted || !asciiEqualFold(attributes.Attributes["type"], "hidden") {
		return nil, false, nil
	}

	input, err := p.parseNode(parent)
	if err != nil {
		return nil, true, err
	}
	return input, true, nil
}

func asciiEqualFold(value, expected string) bool {
	if len(value) != len(expected) {
		return false
	}
	for index := 0; index < len(value); index++ {
		left := value[index]
		if left >= 'A' && left <= 'Z' {
			left += 'a' - 'A'
		}
		if left != expected[index] {
			return false
		}
	}
	return true
}

func asciiLower(value string) string {
	var lowered strings.Builder
	lowered.Grow(len(value))
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character >= 'A' && character <= 'Z' {
			character += 'a' - 'A'
		}
		lowered.WriteByte(character)
	}
	return lowered.String()
}

func (p *HTMLParser) nextTableTokenReconstructsFormatting(parent *types.Node) bool {
	if !p.nextTokenReconstructsFormatting() {
		return false
	}
	if _, _, tag := p.peekTagName(); tag {
		return true
	}
	// The in-table-text rules buffer character tokens before deciding whether
	// foster parenting applies. A buffer containing only HTML whitespace stays
	// in the table and must not reconstruct an off-stack formatting element.
	preview := *p
	previewParent := &types.Node{Type: types.ElementNode, Name: parent.Name, Children: []*types.Node{}}
	text, err := preview.parseTextNode(previewParent, preview.pos, preview.line, preview.col)
	if err != nil || text == nil {
		return true
	}
	return !isHTMLWhitespaceString(text.Value)
}

func (p *HTMLParser) parseTableColgroup(table *types.Node, synthetic bool) (*types.Node, error) {
	var group *types.Node
	if synthetic {
		group = syntheticTableNode("colgroup", table)
	} else {
		var err error
		group, err = p.parseTableOpenElement(table)
		if err != nil {
			return nil, err
		}
	}
	for p.pos < len(p.content) {
		if p.currentTemplateEndCloses(group) {
			p.finishTableImplicit(group, group.ContentStart)
			return group, nil
		}
		name, closing, tag := p.peekTagName()
		if tag && !p.currentTagTokenWillEmit() {
			if _, err := p.consumeDiscardedTableToken(group); err != nil {
				return nil, err
			}
			p.finishTableImplicit(group, group.ContentStart)
			return group, nil
		}
		if tag && closing && name == "colgroup" {
			if synthetic {
				_, _, err := p.parseClosingTag()
				return group, err
			}
			return group, p.closeTableNode(group)
		}
		if tag && closing && name == "col" {
			if _, _, err := p.parseClosingTag(); err != nil {
				return nil, err
			}
			continue
		}
		if tag && !closing && name == "col" {
			col, err := p.parseNode(group)
			if err != nil {
				return nil, err
			}
			p.appendTableChild(group, col)
			continue
		}
		if tag && (isTableStructuralName(name) || closing) {
			p.finishTableImplicit(group, group.ContentStart)
			return group, nil
		}
		if p.peek() == '<' && p.startsMarkupToken() && !p.nextTokenIsComment() && !p.nextTokenIsProcessingInstruction() {
			p.finishTableImplicit(group, group.ContentStart)
			return group, nil
		}
		triggerPos, triggerLine, triggerColumn := p.pos, p.line, p.col
		child, err := p.parseNode(group)
		if err != nil {
			return nil, err
		}
		if child != nil && (child.Type == types.CommentNode || child.Type == types.ProcessingInstructionNode || (child.Type == types.TextNode && isHTMLWhitespaceString(child.Value))) {
			p.appendTableChild(group, child)
			continue
		}
		if child != nil {
			// Non-whitespace character tokens leave colgroup mode and are
			// reprocessed by the table. Text cannot be rolled back, so retain
			// the already-tokenized node as fostered content.
			p.fosterTableChild(table, child)
		}
		p.finishElementAt(group, group.TextContent, group.ContentStart, triggerPos, triggerLine, triggerColumn)
		return group, nil
	}
	p.recoverOpenElementsAtEOF = true
	p.finishTableImplicit(group, group.ContentStart)
	return group, nil
}

func syntheticTableNode(name string, parent *types.Node) *types.Node {
	return &types.Node{Type: types.ElementNode, Name: name, NamespaceURI: htmlElementNamespace, Attributes: make(map[string]string), Children: []*types.Node{}, Parent: parent}
}

func (p *HTMLParser) closeTableNode(node *types.Node) error {
	refreshNodeTextContent(node)
	contentEnd := p.pos
	name, emitted, err := p.parseClosingTag()
	if err != nil {
		return err
	}
	if !emitted || name != node.Name {
		return fmt.Errorf("expected closing tag </%s>", node.Name)
	}
	if node.StartLine == 0 {
		return nil
	}
	node.ContentEnd = contentEnd
	node.EndPos = p.pos
	node.EndLine, node.EndColumn = p.line, p.col
	return nil
}

func (p *HTMLParser) finishTableImplicit(node *types.Node, contentStart int) {
	refreshNodeTextContent(node)
	p.finishImplicitElement(node, node.TextContent, contentStart)
}

func refreshNodeTextContent(node *types.Node) {
	var only string
	contributors := 0
	for _, child := range node.Children {
		switch child.Type {
		case types.TextNode:
			only = child.Value
			contributors++
		case types.ElementNode:
			only = child.TextContent
			contributors++
		}
		if contributors > 1 {
			break
		}
	}
	if contributors <= 1 {
		node.TextContent = only
		return
	}
	var text strings.Builder
	for _, child := range node.Children {
		switch child.Type {
		case types.TextNode:
			text.WriteString(child.Value)
		case types.ElementNode:
			text.WriteString(child.TextContent)
		}
	}
	node.TextContent = text.String()
}

func (p *HTMLParser) appendTableChild(parent, child *types.Node) {
	if child == nil {
		return
	}
	child.Parent = parent
	if child.Type == types.TextNode && len(parent.Children) > 0 && parent.Children[len(parent.Children)-1].Type == types.TextNode {
		mergeTextNodes(parent.Children[len(parent.Children)-1], child)
	} else {
		parent.Children = append(parent.Children, child)
	}
	p.appendDeferredTableFoster(parent, child)
}

func (p *HTMLParser) fosterTableChild(table, child *types.Node) {
	if table == nil || table.Parent == nil || child == nil {
		return
	}
	p.deferredTableFoster[table] = append(p.deferredTableFoster[table], child)
}

func (p *HTMLParser) appendDeferredTableFoster(parent, child *types.Node) string {
	if child == nil || child.Name != "table" {
		return ""
	}
	fosteredNodes := p.deferredTableFoster[child]
	if len(fosteredNodes) == 0 {
		return ""
	}
	tableIndex := len(parent.Children) - 1
	if tableIndex < 0 || parent.Children[tableIndex] != child {
		tableIndex = -1
		for index, candidate := range parent.Children {
			if candidate == child {
				tableIndex = index
				break
			}
		}
	}
	if tableIndex < 0 {
		return ""
	}
	suffix := append([]*types.Node(nil), parent.Children[tableIndex+1:]...)
	parent.Children = parent.Children[:tableIndex]
	var fosteredText strings.Builder
	for index := 0; index < len(fosteredNodes); {
		fostered := fosteredNodes[index]
		if fostered.Type != types.TextNode {
			fostered.Parent = parent
			parent.Children = append(parent.Children, fostered)
			// Adoption surgery performed while the element was logically open in
			// table mode may have produced deferred siblings. They belong at the
			// fostered element's DOM position, before the table, rather than in the
			// document-level adoption finalizer after the table.
			p.appendDeferredAdoption(parent, fostered)
			if fostered.Type == types.ElementNode {
				fosteredText.WriteString(fostered.TextContent)
			}
			index++
			continue
		}
		first := fostered
		last := fostered
		var value strings.Builder
		for index < len(fosteredNodes) && fosteredNodes[index].Type == types.TextNode {
			last = fosteredNodes[index]
			value.WriteString(last.Value)
			index++
		}
		combined := value.String()
		fosteredText.WriteString(combined)
		first.Value = combined
		first.TextContent = combined
		first.EndPos = last.EndPos
		first.EndLine = last.EndLine
		first.EndColumn = last.EndColumn
		first.Parent = parent
		if len(parent.Children) > 0 && parent.Children[len(parent.Children)-1].Type == types.TextNode {
			mergeTextNodes(parent.Children[len(parent.Children)-1], first)
		} else {
			parent.Children = append(parent.Children, first)
		}
	}
	parent.Children = append(parent.Children, child)
	parent.Children = append(parent.Children, suffix...)
	delete(p.deferredTableFoster, child)
	return fosteredText.String()
}

func (p *HTMLParser) appendDeferredListComments(parent, child *types.Node) {
	var comments []*types.Node
	switch {
	case parent == p.documentHTML && child == p.documentBody:
		comments = p.deferredHTMLComments
		p.deferredHTMLComments = nil
	case parent == p.documentRoot && child == p.documentHTML:
		comments = p.deferredRootComments
		p.deferredRootComments = nil
	default:
		return
	}
	for _, comment := range comments {
		comment.Parent = parent
		parent.Children = append(parent.Children, comment)
	}
}

func (p *HTMLParser) parseTableLooseContainer(parent *types.Node, expected string, stop map[string]bool) (*types.Node, error) {
	node, err := p.parseTableOpenElement(parent)
	if err != nil {
		return nil, err
	}
	if expected == "caption" {
		previousFormattingCounts := p.formattingCounts
		previousFormattingFamilies := p.formattingFamilies
		previousFormattingByName := p.formattingByName
		previousFormattingMarkerStart := p.formattingMarkerStart
		p.activeFormatting = append(p.activeFormatting, &formattingEntry{marker: true})
		p.formattingMarkerStart = len(p.activeFormatting)
		p.formattingCounts = make(map[string]int)
		p.formattingFamilies = make(map[string][]*formattingEntry)
		p.formattingByName = make(map[string][]*formattingEntry)
		defer func() {
			p.clearActiveFormattingToMarker()
			p.formattingCounts = previousFormattingCounts
			p.formattingFamilies = previousFormattingFamilies
			p.formattingByName = previousFormattingByName
			p.formattingMarkerStart = previousFormattingMarkerStart
		}()
		defer p.closeFormattingOnElementReturn(node)
	}
	for p.pos < len(p.content) {
		if p.currentTemplateEndCloses(node) {
			p.finishTableImplicit(node, node.ContentStart)
			return node, nil
		}
		name, closing, tag := p.peekTagName()
		if tag && !p.currentTagTokenWillEmit() {
			if _, err := p.consumeDiscardedTableToken(node); err != nil {
				return nil, err
			}
			p.finishTableImplicit(node, node.ContentStart)
			return node, nil
		}
		if consumed, consumeErr := p.consumeForeignIgnoredEnd(node); consumed || consumeErr != nil {
			if consumeErr != nil {
				return nil, consumeErr
			}
			continue
		}
		if expected == "caption" {
			if handled, formErr := p.handleFormToken(node, false); handled || formErr != nil {
				if formErr != nil {
					return nil, formErr
				}
				continue
			}
		}
		if tag && closing && name == expected {
			return node, p.closeTableNode(node)
		}
		if tag && closing && expected == "caption" {
			if name == "table" {
				p.finishTableImplicit(node, node.ContentStart)
				return node, nil
			}
			if isCaptionIgnoredEndTag(name) {
				_, _, err := p.parseClosingTag()
				if err != nil {
					return nil, err
				}
				continue
			}
		}
		if tag && !closing && stop[name] {
			p.finishTableImplicit(node, node.ContentStart)
			return node, nil
		}
		if expected == "caption" {
			if name, closing, tag := p.peekTagName(); tag && closing && isCoreAdoptionSubject(name) && p.activeFormattingEntryByName(name) == nil {
				start := p.pos
				if _, _, closeErr := p.parseClosingTag(); closeErr != nil {
					return nil, closeErr
				}
				p.extendPreviousTextRange(node, start)
				continue
			}
			if name, closing, tag := p.peekTagName(); tag && closing && isCoreAdoptionSubject(name) {
				action, adoptionErr := p.handleCoreAdoption(node, name, node.TextContent, node.ContentStart, true)
				if adoptionErr != nil {
					return nil, adoptionErr
				}
				if action != adoptionNone {
					continue
				}
			}
		}
		if expected == "caption" {
			p.tableCaptionMode++
		}
		insertionParent := node
		if expected == "caption" {
			insertionParent = p.formattingInsertionParent(node, p.nextTokenReconstructsFormatting())
		}
		child, childErr := p.parseNode(insertionParent)
		if expected == "caption" {
			p.tableCaptionMode--
		}
		if childErr != nil {
			return nil, childErr
		}
		if child != nil {
			if insertionParent != node {
				p.appendFormattingChild(node, insertionParent, child)
				refreshNodeTextContent(node)
			} else {
				p.appendTableChild(node, child)
			}
			p.appendDeferredAdoption(node, child)
		}
	}
	p.recoverOpenElementsAtEOF = true
	p.finishTableImplicit(node, node.ContentStart)
	return node, nil
}

func mergeTextNodes(target, source *types.Node) {
	target.Value += source.Value
	target.TextContent += source.TextContent
	target.EndPos = source.EndPos
	target.EndLine = source.EndLine
	target.EndColumn = source.EndColumn
}

// consumeVoidElementClosingTag applies browser recovery for end tags belonging
// to HTML void elements. Browsers ignore these tokens except for </br>, which
// creates a br element. Any recovered node retains the original token's range.
func (p *HTMLParser) consumeVoidElementClosingTag(parent *types.Node) (*types.Node, bool, error) {
	tagName, closing, ok := p.peekTagName()
	if !ok || !closing || !p.isSelfClosingTag(tagName) {
		return nil, false, nil
	}

	startPos := p.pos
	startLine := p.line
	startColumn := p.col
	if _, emitted, err := p.parseClosingTag(); err != nil {
		return nil, true, err
	} else if !emitted {
		p.recoverOpenElementsAtEOF = true
		p.extendPreviousTextRange(parent, startPos)
		return nil, true, nil
	}
	if tagName != "br" {
		return nil, true, nil
	}

	recovered := &types.Node{
		Type:           types.ElementNode,
		Name:           "br",
		Attributes:     make(map[string]string),
		AttributeOrder: []string{},
		Children:       []*types.Node{},
		Parent:         parent,
		StartPos:       startPos,
		EndPos:         p.pos,
		ContentStart:   p.pos,
		ContentEnd:     p.pos,
		StartLine:      startLine,
		StartColumn:    startColumn,
		EndLine:        p.line,
		EndColumn:      p.col,
	}
	if p.foreignBRReprocess {
		p.foreignBRReprocess = false
		recovered.StartPos, recovered.EndPos = 0, 0
		recovered.ContentStart, recovered.ContentEnd = 0, 0
		recovered.StartLine, recovered.StartColumn = 0, 0
		recovered.EndLine, recovered.EndColumn = 0, 0
	}
	return recovered, true, nil
}

// shouldImplicitlyCloseTableRow reports whether the token at the current
// position closes an open tr according to HTML's "in row" insertion mode.
func (p *HTMLParser) shouldImplicitlyCloseTableRow(currentName string) bool {
	if currentName != "tr" {
		return false
	}

	tagName, closing, ok := p.peekTagName()
	if !ok {
		return false
	}

	if closing {
		switch tagName {
		case "table", "tbody", "tfoot", "thead":
			return true
		default:
			return false
		}
	}

	switch tagName {
	case "caption", "col", "colgroup", "tbody", "tfoot", "thead", "tr":
		return true
	default:
		return false
	}
}

// peekTagName inspects an opening or closing tag without advancing the parser.
func (p *HTMLParser) peekTagName() (name string, closing bool, ok bool) {
	if p.tagPreviewPos == p.pos {
		return p.tagPreviewName, p.tagPreviewClosing, p.tagPreviewOK
	}
	if p.pos >= len(p.content) || p.content[p.pos] != '<' {
		return "", false, false
	}

	position := p.pos + 1
	if position < len(p.content) && p.content[position] == '/' {
		closing = true
		position++
	}
	start := position
	for position < len(p.content) && !isWhitespace(p.content[position]) && p.content[position] != '/' && p.content[position] != '>' {
		position++
	}
	if position == start {
		return "", false, false
	}

	var normalized strings.Builder
	for _, r := range p.content[start:position] {
		if r == 0 {
			r = '\uFFFD'
		} else if isASCIIUpper(r) {
			r += 'a' - 'A'
		}
		normalized.WriteRune(r)
	}
	name = normalized.String()
	p.tagPreviewPos = p.pos
	p.tagPreviewName = name
	p.tagPreviewClosing = closing
	p.tagPreviewOK = true
	p.tagPreviewEmissionKnown = false
	return name, closing, true
}

// consumeInitialPreformattedLF applies the in-body one-shot rule for HTML
// pre/listing elements. The check survives malformed input that emits no
// token, but any emitted token (including a NUL later ignored by the tree
// builder) consumes it. A stripped LF is removed before active-formatting
// reconstruction can create an otherwise empty wrapper.
func (p *HTMLParser) consumeInitialPreformattedLF(active *bool) (suppressed, retainSourceStart bool, start, line, column int) {
	if active == nil || !*active || p.pos >= len(p.content) {
		return false, false, 0, 0, 0
	}
	start, line, column = p.pos, p.line, p.col
	if p.peek() == '<' && p.startsMarkupToken() {
		// The missing end-tag-name state for exact </> consumes source input but
		// emits no token, so the next real token still gets the LF check.
		if p.isMissingEndTagName() {
			return false, false, start, line, column
		}
		if _, _, ok := p.peekTagName(); ok {
			if p.currentTagTokenWillEmit() {
				*active = false
			}
			return false, false, start, line, column
		}
		// Comments, doctypes, bogus comments, and processing instructions are
		// emitted tokenizer tokens even when their tree-builder action ignores
		// them. An incomplete token cannot be followed by a later LF anyway.
		*active = false
		return false, false, start, line, column
	}

	// The next token is a character token. It always consumes the one-shot
	// check, even when the data-state NUL rule later omits it from the DOM.
	*active = false
	if p.peek() == 0 {
		return false, false, start, line, column
	}
	preview := *p
	if value, consumed := preview.consumeNumericCharacterReference(); consumed {
		if value != '\n' {
			return false, false, start, line, column
		}
		_, _ = p.consumeNumericCharacterReference()
		suppressed = true
	} else {
		preview = *p
		if value, consumed := preview.consumeNamedCharacterReference(false); consumed {
			if value != "\n" {
				return false, false, start, line, column
			}
			_, _ = p.consumeNamedCharacterReference(false)
			suppressed = true
		} else {
			r, size := p.peekHTMLRune()
			if r != '\n' || size == 0 {
				return false, false, start, line, column
			}
			p.advanceRune(size)
			suppressed = true
		}
	}

	// parse5 retains the stripped token in the source range when the next
	// emitted character is also LF, even though only the latter reaches DOM.
	if p.pos < len(p.content) && (p.peek() != '<' || !p.startsMarkupToken()) && p.peek() != 0 {
		preview = *p
		if value, consumed := preview.consumeNumericCharacterReference(); consumed {
			retainSourceStart = value == '\n'
		} else {
			preview = *p
			if value, consumed := preview.consumeNamedCharacterReference(false); consumed {
				retainSourceStart = strings.HasPrefix(value, "\n")
			} else {
				r, _ := p.peekHTMLRune()
				retainSourceStart = r == '\n'
			}
		}
	}
	return suppressed, retainSourceStart, start, line, column
}

// preformattedRangeStartContinues reports whether a stripped initial LF still
// belongs to the source range of the next emitted LF character token. Exact
// </> consumes input without emitting a token and therefore preserves the
// tentative boundary; every other emitted/non-LF token retires it.
func (p *HTMLParser) preformattedRangeStartContinues() bool {
	if p.pos >= len(p.content) {
		return false
	}
	if p.peek() == '<' && p.startsMarkupToken() {
		return p.isMissingEndTagName()
	}
	if p.peek() == 0 {
		return false
	}
	preview := *p
	if value, consumed := preview.consumeNumericCharacterReference(); consumed {
		return value == '\n'
	}
	preview = *p
	if value, consumed := preview.consumeNamedCharacterReference(false); consumed {
		return strings.HasPrefix(value, "\n")
	}
	r, _ := p.peekHTMLRune()
	return r == '\n'
}

// parseTextNode parses a text node
func (p *HTMLParser) parseTextNode(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	text := ""

	for p.pos < len(p.content) {
		if p.peek() == '<' && p.startsMarkupToken() {
			break
		}
		if p.content[p.pos:] == "</" {
			p.recoverOpenElementsAtEOF = true
		}
		// The HTML tokenizer ignores NUL in the data state. Other states such
		// as comments, attributes, and raw text replace it with U+FFFD.
		if p.peek() == 0 {
			p.advance()
			if text == "" {
				startPos, startLine, startCol = p.pos, p.line, p.col
			}
			continue
		}
		if value, consumed := p.consumeNumericCharacterReference(); consumed {
			text += string(value)
			continue
		}
		if value, consumed := p.consumeNamedCharacterReference(false); consumed {
			text += value
			continue
		}
		// Handle UTF-8 correctly by reading the full character
		r, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		text += string(r)
		p.advanceRune(size)
	}

	if text == "" {
		return nil, nil
	}

	// Preserve all text nodes including whitespace-only text for XPath compatibility
	// Don't skip whitespace-only text nodes as they are significant for XPath expressions

	return &types.Node{
		Type:        types.TextNode,
		Name:        "#text",
		Value:       text, // Preserve original text with whitespace
		TextContent: text, // Preserve original text with whitespace
		Parent:      parent,
		StartPos:    startPos,
		EndPos:      p.pos,
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     p.line,
		EndColumn:   p.col,
	}, nil
}

const (
	htmlElementNamespace = "http://www.w3.org/1999/xhtml"
	svgElementNamespace  = "http://www.w3.org/2000/svg"
	mathElementNamespace = "http://www.w3.org/1998/Math/MathML"
)

var svgTagNameAdjustments = map[string]string{
	"altglyph": "altGlyph", "altglyphdef": "altGlyphDef", "altglyphitem": "altGlyphItem",
	"animatecolor": "animateColor", "animatemotion": "animateMotion", "animatetransform": "animateTransform",
	"clippath": "clipPath", "feblend": "feBlend", "fecolormatrix": "feColorMatrix",
	"fecomponenttransfer": "feComponentTransfer", "fecomposite": "feComposite",
	"feconvolvematrix": "feConvolveMatrix", "fediffuselighting": "feDiffuseLighting",
	"fedisplacementmap": "feDisplacementMap", "fedistantlight": "feDistantLight", "feflood": "feFlood",
	"fedropshadow": "feDropShadow",
	"fefunca":      "feFuncA", "fefuncb": "feFuncB", "fefuncg": "feFuncG", "fefuncr": "feFuncR",
	"fegaussianblur": "feGaussianBlur", "feimage": "feImage", "femerge": "feMerge",
	"femergenode": "feMergeNode", "femorphology": "feMorphology", "feoffset": "feOffset",
	"fepointlight": "fePointLight", "fespecularlighting": "feSpecularLighting", "fespotlight": "feSpotLight",
	"fetile": "feTile", "feturbulence": "feTurbulence", "foreignobject": "foreignObject",
	"glyphref": "glyphRef", "lineargradient": "linearGradient", "radialgradient": "radialGradient",
	"textpath": "textPath",
}

var svgAttributeAdjustments = func() map[string]string {
	names := []string{
		"attributeName", "attributeType", "baseFrequency", "baseProfile", "calcMode", "clipPathUnits",
		"diffuseConstant", "edgeMode", "filterUnits", "glyphRef", "gradientTransform", "gradientUnits",
		"kernelMatrix", "kernelUnitLength", "keyPoints", "keySplines", "keyTimes", "lengthAdjust",
		"limitingConeAngle", "markerHeight", "markerUnits", "markerWidth", "maskContentUnits", "maskUnits",
		"numOctaves", "pathLength", "patternContentUnits", "patternTransform", "patternUnits", "pointsAtX",
		"pointsAtY", "pointsAtZ", "preserveAlpha", "preserveAspectRatio", "primitiveUnits", "refX", "refY",
		"repeatCount", "repeatDur", "requiredExtensions", "requiredFeatures", "specularConstant",
		"specularExponent", "spreadMethod", "startOffset", "stdDeviation", "stitchTiles", "surfaceScale",
		"systemLanguage", "tableValues", "targetX", "targetY", "textLength", "viewBox", "viewTarget",
		"xChannelSelector", "yChannelSelector", "zoomAndPan",
	}
	adjustments := make(map[string]string, len(names))
	for _, name := range names {
		adjustments[strings.ToLower(name)] = name
	}
	return adjustments
}()

func adjustSVGTagName(name string) string {
	if adjusted := svgTagNameAdjustments[name]; adjusted != "" {
		return adjusted
	}
	// parseTagName has already applied the tokenizer's ASCII-only folding.
	// Preserve non-ASCII code points exactly; Unicode lowercasing would change
	// foreign element identity in a way browsers do not.
	return name
}

func adjustSVGAttributes(node *types.Node) {
	if node == nil || len(node.AttributeOrder) == 0 {
		return
	}
	for index, oldName := range node.AttributeOrder {
		adjusted := svgAttributeAdjustments[oldName]
		if adjusted == "" || adjusted == oldName {
			continue
		}
		value := node.Attributes[oldName]
		delete(node.Attributes, oldName)
		node.Attributes[adjusted] = value
		node.AttributeOrder[index] = adjusted
	}
}

func adjustMathMLAttributes(node *types.Node) {
	if node == nil || len(node.AttributeOrder) == 0 {
		return
	}
	for index, oldName := range node.AttributeOrder {
		if oldName != "definitionurl" {
			continue
		}
		value := node.Attributes[oldName]
		delete(node.Attributes, oldName)
		node.Attributes["definitionURL"] = value
		node.AttributeOrder[index] = "definitionURL"
	}
}

const (
	xlinkAttributeNamespace = "http://www.w3.org/1999/xlink"
	xmlAttributeNamespace   = "http://www.w3.org/XML/1998/namespace"
	xmlnsAttributeNamespace = "http://www.w3.org/2000/xmlns/"
)

func initializeAttributeMetadata(node *types.Node, foreign bool) {
	if node == nil {
		return
	}
	node.AttributeNamespaces = make(map[string]string, len(node.AttributeOrder))
	node.AttributeLocalNames = make(map[string]string, len(node.AttributeOrder))
	node.AttributePrefixes = make(map[string]string, len(node.AttributeOrder))
	for _, name := range node.AttributeOrder {
		local, prefix, namespace := name, "", ""
		if foreign {
			switch name {
			case "xlink:actuate", "xlink:arcrole", "xlink:href", "xlink:role", "xlink:show", "xlink:title", "xlink:type":
				local, prefix, namespace = strings.TrimPrefix(name, "xlink:"), "xlink", xlinkAttributeNamespace
			case "xml:lang", "xml:space":
				local, prefix, namespace = strings.TrimPrefix(name, "xml:"), "xml", xmlAttributeNamespace
			case "xmlns":
				local, namespace = "xmlns", xmlnsAttributeNamespace
			case "xmlns:xlink":
				local, prefix, namespace = "xlink", "xmlns", xmlnsAttributeNamespace
			}
		}
		node.AttributeNamespaces[name] = namespace
		node.AttributeLocalNames[name] = local
		node.AttributePrefixes[name] = prefix
	}
}

func (p *HTMLParser) finishForeignSelfClosing(node *types.Node) {
	node.ContentStart = p.pos
	node.ContentEnd = p.pos
	node.EndPos = p.pos
	node.EndLine = p.line
	node.EndColumn = p.col
}

func (p *HTMLParser) parseSVGElement(node *types.Node, contentStart int) (*types.Node, error) {
	key := asciiLower(node.Name)
	previous, hadPrevious := p.activeSVGNames[key]
	p.activeSVGNames[key] = node
	defer func() {
		if hadPrevious {
			p.activeSVGNames[key] = previous
		} else {
			delete(p.activeSVGNames, key)
		}
	}()

	var text strings.Builder
	for p.pos < len(p.content) {
		if p.currentTemplateEndCloses(node) {
			p.finishImplicitElement(node, text.String(), contentStart)
			return node, nil
		}
		if p.templateForeignTransition || (isForeignIntegrationOwner(node) && p.currentTemplateTableTransitionWillEmit(node)) {
			p.templateForeignTransition = true
			p.finishImplicitElement(node, text.String(), contentStart)
			return node, nil
		}
		if isForeignIntegrationOwner(node) {
			handled, err := p.processForeignIntegrationToken(node, &text)
			if err != nil {
				return nil, err
			}
			if handled {
				continue
			}
		}
		if p.content[p.pos] == '<' && p.pos+1 < len(p.content) && p.content[p.pos+1] == '/' && p.startsMarkupToken() {
			tokenStart, tokenLine, tokenColumn := p.pos, p.line, p.col
			if p.isMissingEndTagName() {
				p.advanceRune(3)
				p.extendPreviousTextRange(node, tokenStart)
				continue
			}
			if p.isBogusEndTagOpen() {
				comment, err := p.parseBogusEndTagComment(node, tokenStart, tokenLine, tokenColumn)
				if err != nil {
					return nil, err
				}
				p.appendSVGChild(node, comment, &text)
				continue
			}
			name, _, ok := p.peekTagName()
			if !ok {
				child, err := p.parseSVGText(node)
				if err != nil {
					return nil, err
				}
				p.appendSVGChild(node, child, &text)
				continue
			}
			if (name == "p" || name == "br") && p.currentEndTagTokenWillEmit() {
				if name == "br" {
					p.foreignBRReprocess = true
				}
				p.addForeignIgnoredEnd(key)
				p.finishElementAt(node, text.String(), contentStart, tokenStart, tokenLine, tokenColumn)
				return node, nil
			}
			foreignTarget := p.activeSVGNames[asciiLower(name)]
			if foreignTarget != nil && foreignTarget != node {
				p.addForeignIgnoredEnd(key)
				p.finishElementAt(node, text.String(), contentStart, tokenStart, tokenLine, tokenColumn)
				return node, nil
			}
			if p.foreignIgnoredEnds[name] > 0 && foreignTarget == nil && !asciiEqualFold(name, key) {
				start := p.pos
				if _, emitted, err := p.parseClosingTag(); err != nil {
					return nil, err
				} else if !emitted {
					p.extendPreviousTextRange(node, start)
					p.recoverOpenElementsAtEOF = true
				}
				p.retireForeignIgnoredEnd(name)
				continue
			}
			closingName, emitted, err := p.parseClosingTag()
			if err != nil {
				return nil, err
			}
			if !emitted {
				p.extendPreviousTextRange(node, tokenStart)
				p.recoverOpenElementsAtEOF = true
				p.finishImplicitElement(node, text.String(), contentStart)
				return node, nil
			}
			if asciiEqualFold(closingName, key) {
				node.TextContent = text.String()
				node.ContentStart = contentStart
				node.ContentEnd = tokenStart
				node.EndPos = p.pos
				node.EndLine = p.line
				node.EndColumn = p.col
				return node, nil
			}
			// Foreign end tags without a matching SVG ancestor are parse errors
			// that do not alter the tree. A complete ignored token does not extend
			// a terminal text node; if more text follows, normal coalescing spans
			// the ignored source gap.
			continue
		}

		startPos, startLine, startColumn := p.pos, p.line, p.col
		if p.svgStartTokenBreaksOut() {
			p.addForeignIgnoredEnd(key)
			p.finishElementAt(node, text.String(), contentStart, startPos, startLine, startColumn)
			return node, nil
		}
		switch {
		case strings.HasPrefix(p.content[p.pos:], "<!--"):
			comment, err := p.parseComment(node, startPos, startLine, startColumn)
			if err != nil {
				return nil, err
			}
			p.appendSVGChild(node, comment, &text)
		case strings.HasPrefix(p.content[p.pos:], "<![CDATA["):
			cdata := p.parseSVGCDATA(node, startPos, startLine, startColumn)
			p.appendSVGChild(node, cdata, &text)
		case hasASCIIPrefixFold(p.content[p.pos:], "<!DOCTYPE"):
			if _, err := p.parseDoctype(node, startPos, startLine, startColumn); err != nil {
				return nil, err
			}
			p.extendPreviousTextRange(node, startPos)
		case strings.HasPrefix(p.content[p.pos:], "<?"):
			instruction, err := p.parseProcessingInstruction(node, startPos, startLine, startColumn)
			if err != nil {
				return nil, err
			}
			p.appendSVGChild(node, instruction, &text)
		case strings.HasPrefix(p.content[p.pos:], "<!"):
			comment, err := p.parseBogusComment(node, startPos, startLine, startColumn)
			if err != nil {
				return nil, err
			}
			p.appendSVGChild(node, comment, &text)
		case p.peek() == '<' && p.startsMarkupToken():
			child, emitted, err := p.parseSVGStartElement(node, startPos, startLine, startColumn)
			if err != nil {
				return nil, err
			}
			if !emitted {
				p.extendPreviousTextRange(node, startPos)
				p.recoverOpenElementsAtEOF = true
				p.finishImplicitElement(node, text.String(), contentStart)
				return node, nil
			}
			p.appendSVGChild(node, child, &text)
		default:
			child, err := p.parseSVGText(node)
			if err != nil {
				return nil, err
			}
			p.appendSVGChild(node, child, &text)
		}
	}

	p.recoverOpenElementsAtEOF = true
	p.finishImplicitElement(node, text.String(), contentStart)
	return node, nil
}

func isSVGHTMLIntegrationPoint(node *types.Node) bool {
	if node == nil || node.NamespaceURI != svgElementNamespace {
		return false
	}
	switch node.Name {
	case "foreignObject", "desc", "title":
		return true
	default:
		return false
	}
}

func isMathMLTextIntegrationPoint(node *types.Node) bool {
	if node == nil || node.NamespaceURI != mathElementNamespace {
		return false
	}
	switch node.Name {
	case "mi", "mo", "mn", "ms", "mtext":
		return true
	default:
		return false
	}
}

func isMathMLAnnotationXML(node *types.Node) bool {
	return node != nil && node.NamespaceURI == mathElementNamespace && node.Name == "annotation-xml"
}

func isMathMLAnnotationXMLHTMLIntegrationPoint(node *types.Node) bool {
	if !isMathMLAnnotationXML(node) {
		return false
	}
	encoding, ok := node.Attributes["encoding"]
	return ok && (asciiEqualFold(encoding, "text/html") || asciiEqualFold(encoding, "application/xhtml+xml"))
}

func isForeignIntegrationOwner(node *types.Node) bool {
	return isSVGHTMLIntegrationPoint(node) || isMathMLTextIntegrationPoint(node) || isMathMLAnnotationXML(node)
}

func (p *HTMLParser) processForeignIntegrationToken(parent *types.Node, text *strings.Builder) (bool, error) {
	if p.pos >= len(p.content) {
		return false, nil
	}
	textIntegration := isMathMLTextIntegrationPoint(parent)
	annotationIntegration := isMathMLAnnotationXMLHTMLIntegrationPoint(parent)
	htmlIntegration := isSVGHTMLIntegrationPoint(parent) || textIntegration || annotationIntegration
	if htmlIntegration && (p.peek() != '<' || !p.startsMarkupToken()) {
		startPos, startLine, startColumn := p.pos, p.line, p.col
		child, err := p.parseTextNode(parent, startPos, startLine, startColumn)
		if err != nil {
			return true, err
		}
		p.appendSVGChild(parent, child, text)
		return true, nil
	}
	name, closing, tag := p.peekTagName()
	if htmlIntegration && closing && (name == "p" || name == "br") && p.currentEndTagTokenWillEmit() {
		if name == "br" {
			p.foreignBRReprocess = true
		}
		child, err := p.parseNode(parent)
		if err != nil {
			return true, err
		}
		p.appendSVGChild(parent, child, text)
		return true, nil
	}
	if !tag || closing {
		return false, nil
	}
	if textIntegration && (name == "mglyph" || name == "malignmark") {
		return false, nil
	}
	if isMathMLAnnotationXML(parent) && !annotationIntegration && name != "svg" {
		return false, nil
	}
	if handled, err := p.handleDocumentStartToken(); handled || err != nil {
		return handled, err
	}

	previousNames := p.activeSVGNames
	if name == "svg" || name == "math" {
		// An HTML integration point is a foreign-scope barrier. A nested SVG
		// or MathML island must not be able to match an end tag against the outer
		// stack through the intervening HTML insertion context.
		p.activeSVGNames = make(map[string]*types.Node)
		defer func() { p.activeSVGNames = previousNames }()
	}
	p.foreignHTMLDepth++
	child, err := p.parseNode(parent)
	p.foreignHTMLDepth--
	if err != nil {
		return true, err
	}
	p.appendSVGChild(parent, child, text)
	return true, nil
}

func (p *HTMLParser) currentEndTagTokenWillEmit() bool {
	preview := *p
	_, emitted, err := preview.parseClosingTag()
	return err == nil && emitted
}

func (p *HTMLParser) addForeignIgnoredEnd(name string) {
	p.foreignIgnoredEnds[name]++
	p.foreignIgnoredEndCount++
	p.foreignBreakoutRecovery = true
}

func (p *HTMLParser) retireForeignIgnoredEnd(name string) {
	if p.foreignIgnoredEnds[name] == 0 {
		return
	}
	p.foreignIgnoredEnds[name]--
	if p.foreignIgnoredEndCount > 0 {
		p.foreignIgnoredEndCount--
	}
	if p.foreignIgnoredEndCount == 0 && p.foreignBreakoutBoundary == nil {
		p.foreignBreakoutRecovery = false
	}
}

func (p *HTMLParser) consumeForeignIgnoredEnd(parent *types.Node) (bool, error) {
	name, closing, ok := p.peekTagName()
	if !ok || !closing || p.foreignIgnoredEnds[name] == 0 || nearestOpenElementByName(parent, name) != nil {
		return false, nil
	}
	start := p.pos
	_, emitted, err := p.parseClosingTag()
	if err != nil {
		return true, err
	}
	if !emitted {
		p.extendPreviousTextRange(parent, start)
		p.recoverOpenElementsAtEOF = true
	}
	p.retireForeignIgnoredEnd(name)
	return true, nil
}

func (p *HTMLParser) handleForeignDocumentStart() (bool, error) {
	if p.foreignHTMLDepth == 0 && p.foreignIgnoredEndCount == 0 && !p.foreignBreakoutRecovery {
		return false, nil
	}
	return p.handleDocumentStartToken()
}

func (p *HTMLParser) handleDocumentStartToken() (bool, error) {
	name, closing, ok := p.peekTagName()
	if !ok || closing || (name != "html" && name != "body" && name != "head") || !p.currentTagTokenWillEmit() {
		return false, nil
	}
	switch name {
	case "html":
		return true, p.mergeDuplicateStartInto(p.documentHTML, "html")
	case "body":
		return true, p.mergeDuplicateStartInto(p.documentBody, "body")
	default:
		p.consumeIgnoredStartTag()
		return true, nil
	}
}

func (p *HTMLParser) svgStartTokenBreaksOut() bool {
	name, closing, ok := p.peekTagName()
	if !ok || closing || !p.currentTagTokenWillEmit() {
		return false
	}
	switch name {
	case "b", "big", "blockquote", "body", "br", "center", "code", "dd", "div", "dl", "dt", "em", "embed",
		"h1", "h2", "h3", "h4", "h5", "h6", "head", "hr", "i", "img", "li", "listing", "menu", "meta",
		"nobr", "ol", "p", "pre", "ruby", "s", "small", "span", "strong", "strike", "sub", "sup", "table", "tt", "u", "ul", "var":
		return true
	case "font":
		preview := *p
		preview.advance()
		if preview.parseTagName() != "font" {
			return false
		}
		attributes := &types.Node{Attributes: make(map[string]string)}
		if _, emitted := preview.parseStartTagTail(attributes); !emitted {
			return false
		}
		for _, attribute := range []string{"color", "face", "size"} {
			if _, present := attributes.Attributes[attribute]; present {
				return true
			}
		}
	}
	return false
}

func (p *HTMLParser) matchingForeignAncestorWithinIntegration(node *types.Node, closingName string) (*types.Node, bool) {
	if p.foreignHTMLDepth == 0 || p.activeSVGNames[asciiLower(closingName)] == nil {
		return nil, false
	}
	blocked := false
	for current := node; current != nil; current = current.Parent {
		if current.NamespaceURI == svgElementNamespace || current.NamespaceURI == mathElementNamespace {
			if asciiEqualFold(asciiLower(current.Name), closingName) {
				return current, blocked
			}
			continue
		}
		// An end tag processed in the HTML insertion mode cannot reach back
		// through a live HTML child to close its foreign integration owner.
		// The owner itself is the special-element barrier, even when the live
		// HTML child is ordinary phrasing content such as span, b, or i.
		if current.Type == types.ElementNode {
			blocked = true
		}
	}
	return nil, false
}

func foreignRecoveryGenericEndTarget(node *types.Node, closingName string) *types.Node {
	for current := node; current != nil && current.NamespaceURI == htmlElementNamespace; current = current.Parent {
		if current.Name == closingName {
			return current
		}
		if isHTMLSpecialElement(current.Name) {
			return nil
		}
	}
	return nil
}

func hasASCIIPrefixFold(value, prefix string) bool {
	return len(value) >= len(prefix) && asciiEqualFold(value[:len(prefix)], strings.ToLower(prefix))
}

func (p *HTMLParser) parseSVGStartElement(parent *types.Node, startPos, startLine, startColumn int) (*types.Node, bool, error) {
	p.advance()
	name := p.parseTagName()
	if name == "" {
		return nil, false, fmt.Errorf("expected foreign tag name at position %d", startPos)
	}
	namespace := parent.NamespaceURI
	adjustedName := name
	if namespace == svgElementNamespace {
		adjustedName = adjustSVGTagName(name)
	}
	node := &types.Node{
		Type: types.ElementNode, Name: adjustedName, NamespaceURI: namespace,
		Attributes: make(map[string]string), AttributeOrder: []string{}, Children: []*types.Node{}, Parent: parent,
		StartPos: startPos, StartLine: startLine, StartColumn: startColumn,
	}
	if owner := p.templateOwner[parent]; owner != nil {
		p.templateOwner[node] = owner
	}
	selfClosing, emitted := p.parseStartTagTail(node)
	if !emitted {
		return nil, false, nil
	}
	switch namespace {
	case svgElementNamespace:
		adjustSVGAttributes(node)
	case mathElementNamespace:
		adjustMathMLAttributes(node)
	}
	initializeAttributeMetadata(node, true)
	node.ContentStart = p.pos
	p.elementContentStartLine[node] = p.line
	p.elementContentStartColumn[node] = p.col
	if selfClosing {
		p.finishForeignSelfClosing(node)
		return node, true, nil
	}
	parsed, err := p.parseSVGElement(node, p.pos)
	return parsed, true, err
}

func (p *HTMLParser) parseSVGText(parent *types.Node) (*types.Node, error) {
	startPos, startLine, startColumn := p.pos, p.line, p.col
	var value strings.Builder
	for p.pos < len(p.content) {
		if p.peek() == '<' && p.startsMarkupToken() {
			break
		}
		if reference, consumed := p.consumeNumericCharacterReference(); consumed {
			value.WriteRune(reference)
			continue
		}
		if reference, consumed := p.consumeNamedCharacterReference(false); consumed {
			value.WriteString(reference)
			continue
		}
		r, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		value.WriteRune(r)
		p.advanceRune(size)
	}
	if value.Len() == 0 {
		return nil, nil
	}
	text := value.String()
	return &types.Node{
		Type: types.TextNode, Name: "#text", Value: text, TextContent: text, Parent: parent,
		StartPos: startPos, EndPos: p.pos, StartLine: startLine, StartColumn: startColumn,
		EndLine: p.line, EndColumn: p.col,
	}, nil
}

func (p *HTMLParser) parseSVGCDATA(parent *types.Node, startPos, startLine, startColumn int) *types.Node {
	const prefix = "<![CDATA["
	p.advanceRune(len(prefix))
	var value strings.Builder
	for p.pos < len(p.content) && !strings.HasPrefix(p.content[p.pos:], "]]>") {
		r, size := p.peekHTMLRune()
		value.WriteRune(r)
		p.advanceRune(size)
	}
	if strings.HasPrefix(p.content[p.pos:], "]]>") {
		p.advanceRune(3)
	} else {
		p.recoverOpenElementsAtEOF = true
	}
	text := value.String()
	return &types.Node{
		Type: types.TextNode, Name: "#text", Value: text, TextContent: text, Parent: parent,
		StartPos: startPos, EndPos: p.pos, StartLine: startLine, StartColumn: startColumn,
		EndLine: p.line, EndColumn: p.col,
	}
}

func (p *HTMLParser) appendSVGChild(parent, child *types.Node, text *strings.Builder) {
	if child == nil {
		return
	}
	child.Parent = parent
	if child.Type == types.TextNode && len(parent.Children) > 0 && parent.Children[len(parent.Children)-1].Type == types.TextNode {
		mergeTextNodes(parent.Children[len(parent.Children)-1], child)
	} else {
		parent.Children = append(parent.Children, child)
	}
	if child.Type == types.TextNode || child.Type == types.ElementNode {
		text.WriteString(child.TextContent)
	}
}

// consumeNumericCharacterReference implements the numeric branch of HTML's
// character-reference tokenizer. It consumes only references containing at
// least one digit; malformed "&#" and "&#x" prefixes remain ordinary text.
// Advancing across the original ASCII source keeps decoded DOM values and
// original byte positions independent.
func (p *HTMLParser) consumeNumericCharacterReference() (rune, bool) {
	if p.pos+2 >= len(p.content) || p.content[p.pos:p.pos+2] != "&#" {
		return 0, false
	}

	position := p.pos + 2
	base := uint32(10)
	if position < len(p.content) && (p.content[position] == 'x' || p.content[position] == 'X') {
		base = 16
		position++
	}
	digitsStart := position
	value := uint32(0)
	overflow := false
	for position < len(p.content) {
		digit, ok := numericReferenceDigit(p.content[position], base)
		if !ok {
			break
		}
		if !overflow {
			if value > (0x10FFFF-digit)/base {
				overflow = true
			} else {
				value = value*base + digit
			}
		}
		position++
	}
	if position == digitsStart {
		return 0, false
	}
	if position < len(p.content) && p.content[position] == ';' {
		position++
	}

	p.advanceRune(position - p.pos)
	if overflow || value == 0 || value >= 0xD800 && value <= 0xDFFF {
		return '\uFFFD', true
	}
	if replacement, ok := numericReferenceControlReplacement[value]; ok {
		value = replacement
	}
	return rune(value), true
}

func numericReferenceDigit(character byte, base uint32) (uint32, bool) {
	switch {
	case character >= '0' && character <= '9':
		return uint32(character - '0'), true
	case base == 16 && character >= 'A' && character <= 'F':
		return uint32(character-'A') + 10, true
	case base == 16 && character >= 'a' && character <= 'f':
		return uint32(character-'a') + 10, true
	default:
		return 0, false
	}
}

var numericReferenceControlReplacement = map[uint32]uint32{
	0x80: 0x20AC,
	0x82: 0x201A,
	0x83: 0x0192,
	0x84: 0x201E,
	0x85: 0x2026,
	0x86: 0x2020,
	0x87: 0x2021,
	0x88: 0x02C6,
	0x89: 0x2030,
	0x8A: 0x0160,
	0x8B: 0x2039,
	0x8C: 0x0152,
	0x8E: 0x017D,
	0x91: 0x2018,
	0x92: 0x2019,
	0x93: 0x201C,
	0x94: 0x201D,
	0x95: 0x2022,
	0x96: 0x2013,
	0x97: 0x2014,
	0x98: 0x02DC,
	0x99: 0x2122,
	0x9A: 0x0161,
	0x9B: 0x203A,
	0x9C: 0x0153,
	0x9E: 0x017E,
	0x9F: 0x0178,
}

// consumeNamedCharacterReference decodes the longest named reference from the
// current ampersand. The standard library supplies the complete WHATWG entity
// table, including legacy semicolonless and two-code-point names. Attribute
// values apply HTML's historical ambiguity exception: a semicolonless match
// followed by an ASCII alphanumeric or '=' remains literal.
func (p *HTMLParser) consumeNamedCharacterReference(attribute bool) (string, bool) {
	if p.pos+1 >= len(p.content) || p.content[p.pos] != '&' || p.content[p.pos+1] == '#' {
		return "", false
	}

	nameEnd := p.pos + 1
	// CounterClockwiseContourIntegral; is the longest WHATWG entity name at
	// 32 bytes. No lookup can succeed beyond this bound, so do not scan an
	// untrusted alphanumeric run quadratically.
	const longestNamedCharacterReference = len("CounterClockwiseContourIntegral;")
	lookupLimit := p.pos + 1 + longestNamedCharacterReference
	if lookupLimit > len(p.content) {
		lookupLimit = len(p.content)
	}
	for nameEnd < lookupLimit && isASCIIAlphanumeric(p.content[nameEnd]) {
		nameEnd++
	}
	if nameEnd == p.pos+1 {
		return "", false
	}
	candidateEnd := nameEnd
	if candidateEnd < len(p.content) && p.content[candidateEnd] == ';' {
		candidateEnd++
	}

	// Infer the longest actual table match from UnescapeString while keeping
	// any unmatched suffix out of the consumed range. A longer candidate that
	// merely preserves more suffix has the form previousDecode + rawSuffix;
	// a different result identifies a new, longer entity-table entry.
	matchEnd := 0
	matchValue := ""
	for end := p.pos + 2; end <= candidateEnd; end++ {
		candidate := p.content[p.pos:end]
		decoded := stdhtml.UnescapeString(candidate)
		if decoded == candidate {
			continue
		}
		if matchEnd == 0 || decoded != matchValue+p.content[matchEnd:end] {
			matchEnd = end
			matchValue = decoded
		}
	}
	if matchEnd == 0 {
		return "", false
	}
	matchedSemicolon := p.content[matchEnd-1] == ';'
	if attribute && !matchedSemicolon && matchEnd < len(p.content) && (isASCIIAlphanumeric(p.content[matchEnd]) || p.content[matchEnd] == '=') {
		return "", false
	}

	p.advanceRune(matchEnd - p.pos)
	return matchValue, true
}

func isASCIIAlphanumeric(character byte) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9'
}

// startsMarkupToken reports whether '<' enters a markup tokenizer state. A
// less-than sign followed by any other character remains ordinary data text.
func (p *HTMLParser) startsMarkupToken() bool {
	if p.pos >= len(p.content) || p.content[p.pos] != '<' || p.pos+1 >= len(p.content) {
		return false
	}

	next := p.content[p.pos+1]
	// At EOF, the end-tag-open state emits both '<' and '/' as text.
	if next == '/' && p.pos+2 >= len(p.content) {
		return false
	}
	return isASCIIAlpha(next) || next == '!' || next == '?' || next == '/'
}

func (p *HTMLParser) isMissingEndTagName() bool {
	return p.pos+2 < len(p.content) && p.content[p.pos:p.pos+3] == "</>"
}

func (p *HTMLParser) isBogusEndTagOpen() bool {
	if p.pos+2 >= len(p.content) || p.content[p.pos:p.pos+2] != "</" {
		return false
	}
	next := p.content[p.pos+2]
	return next != '>' && !isASCIIAlpha(next)
}

// parseBogusEndTagComment handles an invalid first character after "</".
// The slash and less-than sign are tokenizer syntax and are not comment data.
func (p *HTMLParser) parseBogusEndTagComment(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	if !p.isBogusEndTagOpen() {
		return nil, fmt.Errorf("expected invalid end-tag opener at position %d", p.pos)
	}

	p.advanceRune(2) // Skip "</".
	var value strings.Builder
	for p.pos < len(p.content) && p.peek() != '>' {
		r, size := p.peekHTMLRune()
		value.WriteRune(r)
		p.advanceRune(size)
	}
	if p.peek() == '>' {
		p.advance()
	} else {
		p.recoverOpenElementsAtEOF = true
	}

	comment := value.String()
	return &types.Node{
		Type:        types.CommentNode,
		Name:        "#comment",
		Value:       comment,
		TextContent: comment,
		Parent:      parent,
		StartPos:    startPos,
		EndPos:      p.pos,
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     p.line,
		EndColumn:   p.col,
	}, nil
}

// parseBogusComment applies HTML's recovery for an incorrectly opened
// comment. Everything after "<!" through the next '>' (or EOF) becomes the
// comment value while the node range remains tied to the original input.
func (p *HTMLParser) parseBogusComment(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	declaration := strings.HasPrefix(p.content[p.pos:], "<!")
	questionMark := strings.HasPrefix(p.content[p.pos:], "<?")
	if !declaration && !questionMark {
		return nil, fmt.Errorf("expected markup declaration at position %d", p.pos)
	}

	p.advance() // Skip '<'
	if declaration {
		p.advance() // Skip '!'
	}

	var comment strings.Builder
	for p.pos < len(p.content) && p.peek() != '>' {
		r, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		comment.WriteRune(r)
		p.advanceRune(size)
	}
	if p.peek() == '>' {
		p.advance()
	} else {
		p.recoverOpenElementsAtEOF = true
	}

	value := comment.String()
	return &types.Node{
		Type:        types.CommentNode,
		Name:        "#comment",
		Value:       value,
		TextContent: value,
		Parent:      parent,
		StartPos:    startPos,
		EndPos:      p.pos,
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     p.line,
		EndColumn:   p.col,
	}, nil
}

func (p *HTMLParser) parseProcessingInstruction(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	if !strings.HasPrefix(p.content[p.pos:], "<?") {
		return nil, fmt.Errorf("expected processing instruction at position %d", p.pos)
	}
	p.advanceRune(2)
	if p.pos >= len(p.content) {
		p.recoverOpenElementsAtEOF = true
		return nil, nil
	}
	if !isPITargetStart(p.peek()) {
		p.pos, p.line, p.col = startPos, startLine, startCol
		return p.parseBogusComment(parent, startPos, startLine, startCol)
	}
	var target strings.Builder
	for p.pos < len(p.content) && isPITargetContinue(p.peek()) {
		character := p.peek()
		if character >= 'A' && character <= 'Z' {
			character += 'a' - 'A'
		}
		target.WriteByte(character)
		p.advance()
	}
	name := target.String()
	if asciiEqualFold(name, "xml") || asciiEqualFold(name, "xml-stylesheet") || (p.pos < len(p.content) && !isPIDelimiter(p.peek())) {
		p.pos, p.line, p.col = startPos, startLine, startCol
		return p.parseBogusComment(parent, startPos, startLine, startCol)
	}
	for p.pos < len(p.content) && isWhitespace(p.peek()) {
		p.advance()
	}
	var data strings.Builder
	for p.pos < len(p.content) {
		if p.peek() == '>' {
			p.advance()
			return processingInstructionNode(parent, name, data.String(), startPos, startLine, startCol, p.pos, p.line, p.col), nil
		}
		if p.peek() == '?' && p.pos+1 < len(p.content) && p.content[p.pos+1] == '>' {
			p.advanceRune(2)
			return processingInstructionNode(parent, name, data.String(), startPos, startLine, startCol, p.pos, p.line, p.col), nil
		}
		r, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		data.WriteRune(r)
		p.advanceRune(size)
	}
	p.recoverOpenElementsAtEOF = true
	return nil, nil
}

func isPITargetStart(character byte) bool {
	return character == '_' || character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z'
}

func isPITargetContinue(character byte) bool {
	return isPITargetStart(character) || character == '-' || character == '_' || character >= '0' && character <= '9'
}

func isPIDelimiter(character byte) bool {
	return isWhitespace(character) || character == '?' || character == '>'
}

func processingInstructionNode(parent *types.Node, name, data string, start, startLine, startColumn, end, endLine, endColumn int) *types.Node {
	return &types.Node{Type: types.ProcessingInstructionNode, Name: name, Value: data, TextContent: data, Parent: parent,
		StartPos: start, EndPos: end, StartLine: startLine, StartColumn: startColumn, EndLine: endLine, EndColumn: endColumn}
}

// parseComment parses an HTML comment
func (p *HTMLParser) parseComment(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	if !strings.HasPrefix(p.content[p.pos:], "<!--") {
		return nil, fmt.Errorf("expected comment at position %d", p.pos)
	}

	for i := 0; i < 4; i++ {
		p.advance() // Skip "<!--"
	}

	const (
		commentStart = iota
		commentStartDash
		commentData
		commentLessThan
		commentLessThanBang
		commentLessThanBangDash
		commentLessThanBangDashDash
		commentEndDash
		commentEnd
		commentEndBang
	)

	state := commentStart
	var comment strings.Builder
	terminated := false
	for !terminated && p.pos < len(p.content) {
		c, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		reconsume := false

		switch state {
		case commentStart:
			switch c {
			case '-':
				state = commentStartDash
			case '>':
				terminated = true
			default:
				state = commentData
				reconsume = true
			}
		case commentStartDash:
			switch c {
			case '-':
				state = commentEnd
			case '>':
				terminated = true
			default:
				comment.WriteByte('-')
				state = commentData
				reconsume = true
			}
		case commentData:
			switch c {
			case '<':
				comment.WriteRune(c)
				state = commentLessThan
			case '-':
				state = commentEndDash
			default:
				comment.WriteRune(c)
			}
		case commentLessThan:
			switch c {
			case '!':
				comment.WriteRune(c)
				state = commentLessThanBang
			case '<':
				comment.WriteRune(c)
			default:
				state = commentData
				reconsume = true
			}
		case commentLessThanBang:
			if c == '-' {
				state = commentLessThanBangDash
			} else {
				state = commentData
				reconsume = true
			}
		case commentLessThanBangDash:
			if c == '-' {
				state = commentLessThanBangDashDash
			} else {
				state = commentEndDash
				reconsume = true
			}
		case commentLessThanBangDashDash:
			state = commentEnd
			reconsume = true
		case commentEndDash:
			if c == '-' {
				state = commentEnd
			} else {
				comment.WriteByte('-')
				state = commentData
				reconsume = true
			}
		case commentEnd:
			switch c {
			case '>':
				terminated = true
			case '!':
				state = commentEndBang
			case '-':
				comment.WriteByte('-')
			default:
				comment.WriteString("--")
				state = commentData
				reconsume = true
			}
		case commentEndBang:
			switch c {
			case '-':
				comment.WriteString("--!")
				state = commentEndDash
			case '>':
				terminated = true
			default:
				comment.WriteString("--!")
				state = commentData
				reconsume = true
			}
		}

		if !reconsume {
			p.advanceRune(size)
		}
	}

	value := comment.String()

	return &types.Node{
		Type:        types.CommentNode,
		Name:        "#comment",
		Value:       value,
		TextContent: value,
		Parent:      parent,
		StartPos:    startPos,
		EndPos:      p.pos,
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     p.line,
		EndColumn:   p.col,
	}, nil
}

// parseDoctype emits the browser's DocumentType shape while consuming the
// declaration through the first '>' or EOF. The detailed public/system
// identifier states affect quirks mode, which this DOM model does not expose,
// but they do not alter the DocumentType name or source extent.
func (p *HTMLParser) parseDoctype(parent *types.Node, startPos, startLine, startCol int) (*types.Node, error) {
	if !strings.HasPrefix(strings.ToUpper(p.content[p.pos:]), "<!DOCTYPE") {
		return nil, fmt.Errorf("expected DOCTYPE at position %d", p.pos)
	}

	for i := 0; i < len("<!DOCTYPE"); i++ {
		p.advance()
	}
	p.skipWhitespace()

	var name strings.Builder
	for p.pos < len(p.content) {
		c := p.peek()
		if isWhitespace(c) || c == '>' {
			break
		}
		r, size := p.peekHTMLRune()
		if isASCIIUpper(r) {
			r += 'a' - 'A'
		}
		name.WriteRune(r)
		p.advanceRune(size)
	}

	// Every DOCTYPE state emits at the first greater-than sign. EOF also emits
	// the in-progress token rather than failing the HTML parse.
	for p.pos < len(p.content) && p.peek() != '>' {
		p.advance()
	}
	terminated := p.peek() == '>'
	if terminated {
		p.advance()
	} else {
		// EOF emits the in-progress token. When tree construction ignores that
		// token inside an element, this signal closes the affected open stack.
		p.recoverOpenElementsAtEOF = true
	}
	if parent != nil && parent.Type == types.DocumentNode && p.shouldKeepDoctype(parent) {
		p.quirksMode = !doctypeUsesStandardsTableRules(p.content[startPos:p.pos], name.String())
	}

	return &types.Node{
		Type:        types.DocumentTypeNode,
		Name:        name.String(),
		Value:       "",
		TextContent: "",
		Parent:      parent,
		StartPos:    startPos,
		EndPos:      p.pos,
		StartLine:   startLine,
		StartColumn: startCol,
		EndLine:     p.line,
		EndColumn:   p.col,
	}, nil
}

func doctypeUsesStandardsTableRules(raw, name string) bool {
	if name != "html" || len(raw) < len("<!doctype html>") {
		return false
	}
	declaration := trimHTMLWhitespace(raw)
	if !strings.HasSuffix(declaration, ">") {
		return false
	}
	inside := trimHTMLWhitespace(declaration[len("<!doctype") : len(declaration)-1])
	if len(inside) < len("html") || !asciiEqualFold(inside[:len("html")], "html") {
		return false
	}
	remainder := trimHTMLWhitespace(inside[len("html"):])
	if remainder == "" {
		return true
	}
	keywordEnd := strings.IndexFunc(remainder, func(r rune) bool { return r <= 0x7f && isWhitespace(byte(r)) })
	if keywordEnd < 0 {
		return false
	}
	keyword, rest := remainder[:keywordEnd], trimHTMLWhitespace(remainder[keywordEnd+1:])
	switch {
	case asciiEqualFold(keyword, "system"):
		systemID, _, ok := consumeQuotedDoctypeIdentifier(rest)
		return ok && !asciiEqualFold(systemID, "http://www.ibm.com/data/dtd/v11/ibmxhtml1-transitional.dtd")
	case asciiEqualFold(keyword, "public"):
		publicID, tail, ok := consumeQuotedDoctypeIdentifier(rest)
		if !ok {
			return false
		}
		tail = trimHTMLWhitespace(tail)
		systemIDPresent := false
		if tail != "" {
			_, tail, ok = consumeQuotedDoctypeIdentifier(tail)
			if !ok || trimHTMLWhitespace(tail) != "" {
				return false
			}
			systemIDPresent = true
		}
		publicID = strings.ToLower(publicID)
		if doctypePublicIDForcesQuirks(publicID) {
			return false
		}
		if strings.HasPrefix(publicID, "-//w3c//dtd xhtml 1.0 frameset//") || strings.HasPrefix(publicID, "-//w3c//dtd xhtml 1.0 transitional//") {
			return true
		}
		if strings.HasPrefix(publicID, "-//w3c//dtd html 4.01 frameset//") || strings.HasPrefix(publicID, "-//w3c//dtd html 4.01 transitional//") {
			return systemIDPresent
		}
		return true
	default:
		return false
	}
}

func trimHTMLWhitespace(value string) string {
	return strings.TrimFunc(value, func(r rune) bool { return r <= 0x7f && isWhitespace(byte(r)) })
}

func consumeQuotedDoctypeIdentifier(value string) (identifier, tail string, ok bool) {
	if len(value) < 2 || (value[0] != '\'' && value[0] != '"') {
		return "", value, false
	}
	if end := strings.IndexByte(value[1:], value[0]); end >= 0 {
		end++
		return value[1:end], value[end+1:], true
	}
	return "", value, false
}

func doctypePublicIDForcesQuirks(publicID string) bool {
	if publicID == "html" || publicID == "-//w3o//dtd w3 html strict 3.0//en//" || publicID == "-/w3c/dtd html 4.0 transitional/en" {
		return true
	}
	for _, prefix := range []string{
		"+//silmaril//dtd html pro v0r11 19970101//", "-//advasoft ltd//dtd html 3.0 aswedit + extensions//",
		"-//as//dtd html 3.0 aswedit + extensions//", "-//ietf//dtd html 2.", "-//ietf//dtd html 3",
		"-//ietf//dtd html level", "-//ietf//dtd html strict level", "-//microsoft//dtd internet explorer 2.0 html", "-//microsoft//dtd internet explorer 3.0 html",
		"-//metrius//dtd metrius presentational//", "-//microsoft//dtd internet explorer 4.0 html", "-//netscape comm. corp.//dtd html",
		"-//netscape comm. corp.//dtd strict html//", "-//o'reilly and associates//dtd html 2.0", "-//o'reilly and associates//dtd html extended 1.0",
		"-//o'reilly and associates//dtd html extended relaxed 1.0", "-//softquad software//dtd hotmetal pro 6.0::19990601::extensions to html 4.0//",
		"-//softquad//dtd hotmetal pro 4.0::19971010::extensions to html 4.0//", "-//spyglass//dtd html 2.0 extended//",
		"-//sq//dtd html 2.0 hotmetal + extensions//", "-//sun microsystems corp.//dtd hotjava html//",
		"-//sun microsystems corp.//dtd hotjava strict html//", "-//w3c//dtd html 3", "-//w3c//dtd html 4.0 frameset//",
		"-//w3c//dtd html 4.0 transitional//", "-//w3c//dtd html experimental 19960712//", "-//w3c//dtd html experimental 970421//", "-//w3c//dtd w3 html//",
		"-//w3o//dtd w3 html 3.0//", "-//webtechs//dtd mozilla html 2.0//", "-//webtechs//dtd mozilla html//",
	} {
		if strings.HasPrefix(publicID, prefix) {
			return true
		}
	}
	return false
}

// shouldKeepDoctype applies the observable part of HTML's insertion modes:
// only the first document-level DOCTYPE before body content is inserted.
// Leading comments do not leave the initial declaration phase.
func (p *HTMLParser) shouldKeepDoctype(parent *types.Node) bool {
	if parent == nil || parent.Type != types.DocumentNode {
		return false
	}
	for _, child := range parent.Children {
		if child.Type != types.CommentNode && child.Type != types.ProcessingInstructionNode {
			return false
		}
	}
	return true
}

func (p *HTMLParser) doctypeWasIgnoredAfterBodyContent(parent *types.Node) bool {
	if parent == nil || parent.Type != types.DocumentNode {
		return false
	}
	for _, child := range parent.Children {
		if child.Type != types.CommentNode && child.Type != types.ProcessingInstructionNode && child.Type != types.DocumentTypeNode {
			return true
		}
	}
	return false
}

type startTagState uint8

const (
	beforeAttributeNameState startTagState = iota
	attributeNameState
	afterAttributeNameState
	beforeAttributeValueState
	attributeValueDoubleQuotedState
	attributeValueSingleQuotedState
	attributeValueUnquotedState
	afterAttributeValueQuotedState
	selfClosingStartTagState
)

// parseStartTagTail implements the HTML tokenizer states from "before
// attribute name" through "self-closing start tag". The returned emitted flag
// is false when EOF discards the incomplete start-tag token.
func (p *HTMLParser) parseStartTagTail(node *types.Node) (selfClosing, emitted bool) {
	state := beforeAttributeNameState
	var attributeName strings.Builder
	var attributeValue strings.Builder

	resetAttribute := func() {
		attributeName.Reset()
		attributeValue.Reset()
	}
	commitAttribute := func() {
		name := attributeName.String()
		if name == "" {
			return
		}
		// The tokenizer removes later duplicate attributes after ASCII-case
		// normalization. Keep both the first value and its document order.
		if _, exists := node.Attributes[name]; !exists {
			node.Attributes[name] = attributeValue.String()
			node.AttributeOrder = append(node.AttributeOrder, name)
		}
	}

	for {
		if p.pos >= len(p.content) {
			return false, false
		}

		switch state {
		case beforeAttributeNameState:
			switch c := p.peek(); {
			case isWhitespace(c):
				p.advance()
			case c == '/':
				p.advance()
				state = selfClosingStartTagState
			case c == '>':
				p.advance()
				return false, true
			default:
				resetAttribute()
				// '=' has special handling in this state: it is the first
				// character of an attribute name, not the value separator.
				if c == '=' {
					attributeName.WriteByte(c)
					p.advance()
				}
				state = attributeNameState
			}

		case attributeNameState:
			switch c := p.peek(); {
			case isWhitespace(c):
				p.advance()
				state = afterAttributeNameState
			case c == '/' || c == '>':
				state = afterAttributeNameState
			case c == '=':
				p.advance()
				state = beforeAttributeValueState
			default:
				r, size := p.peekHTMLRune()
				if isASCIIUpper(r) {
					r += 'a' - 'A'
				}
				attributeName.WriteRune(r)
				p.advanceRune(size)
			}

		case afterAttributeNameState:
			switch c := p.peek(); {
			case isWhitespace(c):
				p.advance()
			case c == '/':
				commitAttribute()
				p.advance()
				state = selfClosingStartTagState
			case c == '=':
				p.advance()
				state = beforeAttributeValueState
			case c == '>':
				commitAttribute()
				p.advance()
				return false, true
			default:
				commitAttribute()
				resetAttribute()
				state = attributeNameState
			}

		case beforeAttributeValueState:
			switch c := p.peek(); {
			case isWhitespace(c):
				p.advance()
			case c == '"':
				p.advance()
				state = attributeValueDoubleQuotedState
			case c == '\'':
				p.advance()
				state = attributeValueSingleQuotedState
			case c == '>':
				commitAttribute()
				p.advance()
				return false, true
			default:
				state = attributeValueUnquotedState
			}

		case attributeValueDoubleQuotedState:
			if p.peek() == '"' {
				commitAttribute()
				p.advance()
				state = afterAttributeValueQuotedState
				continue
			}
			if value, consumed := p.consumeNumericCharacterReference(); consumed {
				attributeValue.WriteRune(value)
				continue
			}
			if value, consumed := p.consumeNamedCharacterReference(true); consumed {
				attributeValue.WriteString(value)
				continue
			}
			r, size := p.peekHTMLRune()
			attributeValue.WriteRune(r)
			p.advanceRune(size)

		case attributeValueSingleQuotedState:
			if p.peek() == '\'' {
				commitAttribute()
				p.advance()
				state = afterAttributeValueQuotedState
				continue
			}
			if value, consumed := p.consumeNumericCharacterReference(); consumed {
				attributeValue.WriteRune(value)
				continue
			}
			if value, consumed := p.consumeNamedCharacterReference(true); consumed {
				attributeValue.WriteString(value)
				continue
			}
			r, size := p.peekHTMLRune()
			attributeValue.WriteRune(r)
			p.advanceRune(size)

		case attributeValueUnquotedState:
			switch c := p.peek(); {
			case isWhitespace(c):
				commitAttribute()
				p.advance()
				state = beforeAttributeNameState
			case c == '>':
				commitAttribute()
				p.advance()
				return false, true
			default:
				if value, consumed := p.consumeNumericCharacterReference(); consumed {
					attributeValue.WriteRune(value)
					continue
				}
				if value, consumed := p.consumeNamedCharacterReference(true); consumed {
					attributeValue.WriteString(value)
					continue
				}
				// Quotes, apostrophes, '<', '=', and '`' are parse errors in
				// this state, but browsers still append them to the value.
				r, size := p.peekHTMLRune()
				attributeValue.WriteRune(r)
				p.advanceRune(size)
			}

		case afterAttributeValueQuotedState:
			switch c := p.peek(); {
			case isWhitespace(c):
				p.advance()
				state = beforeAttributeNameState
			case c == '/':
				p.advance()
				state = selfClosingStartTagState
			case c == '>':
				p.advance()
				return false, true
			default:
				// Missing whitespace: reconsume in before-attribute-name.
				state = beforeAttributeNameState
			}

		case selfClosingStartTagState:
			if p.peek() == '>' {
				p.advance()
				return true, true
			}
			// An unexpected character after '/' cancels the self-closing
			// flag and is reconsumed as the beginning of another attribute.
			state = beforeAttributeNameState
		}
	}
}

// extendPreviousTextRange mirrors parse5's location behavior for an ignored
// start-tag token at EOF. The text value is unchanged; only its original-source
// range covers the discarded bytes.
func (p *HTMLParser) extendPreviousTextRange(parent *types.Node, tagStart int) {
	if parent == nil || len(parent.Children) == 0 {
		return
	}
	previous := parent.Children[len(parent.Children)-1]
	if previous.Type != types.TextNode || previous.EndPos != tagStart {
		return
	}
	previous.EndPos = p.pos
	previous.EndLine = p.line
	previous.EndColumn = p.col
}

// parseTagName consumes the tag-name state. Callers enter this helper only
// after the tag-open state has observed an ASCII alpha character.
func (p *HTMLParser) parseTagName() string {
	var name strings.Builder
	for p.pos < len(p.content) {
		c := p.peek()
		if isWhitespace(c) || c == '/' || c == '>' {
			break
		}
		r, size := p.peekHTMLRune()
		if isASCIIUpper(r) {
			r += 'a' - 'A'
		}
		name.WriteRune(r)
		p.advanceRune(size)
	}
	return name.String()
}

// parseClosingTag tokenizes a syntactically valid end-tag opener. Attributes
// and a trailing solidus are consumed by the same states as a start tag but
// discarded. emitted is false when EOF discards the incomplete token.
func (p *HTMLParser) parseClosingTag() (name string, emitted bool, err error) {
	startPos := p.pos
	if p.pos+1 >= len(p.content) || p.content[p.pos:p.pos+2] != "</" {
		return "", false, fmt.Errorf("expected closing tag at position %d", p.pos)
	}

	p.advanceRune(2) // Skip "</".
	name = p.parseTagName()
	if name == "" {
		return "", false, fmt.Errorf("expected closing tag name at position %d", p.pos)
	}

	discardedAttributes := &types.Node{Attributes: make(map[string]string)}
	_, emitted = p.parseStartTagTail(discardedAttributes)
	if !emitted && p.pos < len(p.content) {
		return "", false, fmt.Errorf("failed to parse closing tag at position %d", startPos)
	}
	if !emitted {
		p.recoverOpenElementsAtEOF = true
	}
	return name, emitted, nil
}

// Helper functions
func (p *HTMLParser) peek() byte {
	if p.pos >= len(p.content) {
		return 0
	}
	return p.content[p.pos]
}

func (p *HTMLParser) advance() {
	if p.pos < len(p.content) {
		c := p.content[p.pos]
		if p.utf8Continuation > 0 {
			p.utf8Continuation--
			p.pos++
			return
		}
		switch c {
		case '\r':
			p.line++
			p.col = 1
			p.lastWasCR = true
		case '\n':
			if !p.lastWasCR {
				p.line++
				p.col = 1
			}
			p.lastWasCR = false
		default:
			if c >= utf8.RuneSelf {
				r, size := utf8.DecodeRuneInString(p.content[p.pos:])
				if size > 1 {
					p.utf8Continuation = size - 1
				}
				if r > 0xFFFF {
					p.col += 2
				} else {
					p.col++
				}
			} else {
				p.col++
			}
			p.lastWasCR = false
		}
		p.pos++
	}
}

// peekHTMLRune applies the input-stream preprocessing used by HTML parsers
// while retaining the number of bytes occupied in the original source.
func (p *HTMLParser) peekHTMLRune() (rune, int) {
	if p.pos >= len(p.content) {
		return 0, 0
	}
	if p.content[p.pos] == '\r' {
		if p.pos+1 < len(p.content) && p.content[p.pos+1] == '\n' {
			return '\n', 2
		}
		return '\n', 1
	}
	if p.content[p.pos] == 0 {
		return '\uFFFD', 1
	}
	return utf8.DecodeRuneInString(p.content[p.pos:])
}

// advanceRune advances the position by the given number of bytes (for a UTF-8 rune)
func (p *HTMLParser) advanceRune(size int) {
	for i := 0; i < size && p.pos < len(p.content); i++ {
		p.advance()
	}
}

func (p *HTMLParser) skipWhitespace() {
	for p.pos < len(p.content) {
		c := p.peek()
		if !isWhitespace(c) {
			break
		}
		p.advance()
	}
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

func isHTMLWhitespaceString(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		switch character {
		case ' ', '\t', '\n', '\r', '\f':
		default:
			return false
		}
	}
	return true
}

func isASCIIAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isASCIIUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func (p *HTMLParser) isSelfClosingTag(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "frame", "hr", "img", "input", "keygen", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

// isRawTextElement checks if an element should have its content parsed as raw text
func (p *HTMLParser) isRawTextElement(name string) bool {
	switch name {
	case "script", "style", "textarea", "title", "xmp", "iframe", "noembed", "noframes":
		return true
	default:
		return false
	}
}

func (p *HTMLParser) parsePlaintextContent() textStateContent {
	result := textStateContent{start: p.pos, startLine: p.line, startColumn: p.col}
	var content strings.Builder
	for p.pos < len(p.content) {
		r, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		content.WriteRune(r)
		p.advanceRune(size)
	}
	result.content = content.String()
	result.end = p.pos
	result.endLine, result.endColumn = p.line, p.col
	p.recoverOpenElementsAtEOF = true
	return result
}

type textStateContent struct {
	content                string
	start, end             int
	startLine, startColumn int
	endLine, endColumn     int
}

// parseRawTextContentWithPos parses RCDATA, RAWTEXT, and the simple script-data
// path while retaining the source boundaries of the emitted character tokens.
func (p *HTMLParser) parseRawTextContentWithPos(tagName string) (textStateContent, error) {
	if tagName == "script" {
		return p.parseScriptDataContent()
	}
	var content strings.Builder
	rcdata := tagName == "title" || tagName == "textarea"
	result := textStateContent{start: p.pos, startLine: p.line, startColumn: p.col}
	stripTextareaLF := tagName == "textarea"
	strippedStart := result.start
	strippedStartLine, strippedStartColumn := result.startLine, result.startColumn

	appendRune := func(value rune) {
		if content.Len() == 0 && value == '\n' && result.start > strippedStart {
			// parse5 includes both original newline tokens in the source range
			// when stripping the first still leaves a leading newline.
			result.start = strippedStart
			result.startLine, result.startColumn = strippedStartLine, strippedStartColumn
		}
		content.WriteRune(value)
	}
	appendString := func(value string) {
		if value == "" {
			return
		}
		first, size := utf8.DecodeRuneInString(value)
		appendRune(first)
		content.WriteString(value[size:])
	}
	consumeTextareaInitialLF := func(value string) (string, bool) {
		if !stripTextareaLF {
			return value, false
		}
		stripTextareaLF = false
		if strings.HasPrefix(value, "\n") {
			result.start = p.pos
			result.startLine, result.startColumn = p.line, p.col
			return strings.TrimPrefix(value, "\n"), true
		}
		return value, false
	}

	for p.pos < len(p.content) {
		if p.isAppropriateTextStateEndTag(tagName) {
			contentEnd := p.pos
			contentEndLine, contentEndColumn := p.line, p.col
			_, emitted, err := p.parseClosingTag()
			if err != nil {
				return textStateContent{}, err
			}
			if emitted {
				result.content = content.String()
				result.end = contentEnd
				result.endLine, result.endColumn = contentEndLine, contentEndColumn
				return result, nil
			}
		}
		if rcdata {
			if value, consumed := p.consumeNumericCharacterReference(); consumed {
				decoded, stripped := consumeTextareaInitialLF(string(value))
				if stripped && decoded == "" {
					continue
				}
				appendString(decoded)
				continue
			}
			if value, consumed := p.consumeNamedCharacterReference(false); consumed {
				decoded, stripped := consumeTextareaInitialLF(value)
				if stripped && decoded == "" {
					continue
				}
				appendString(decoded)
				continue
			}
		}
		r, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		p.advanceRune(size)
		decoded, stripped := consumeTextareaInitialLF(string(r))
		if stripped && decoded == "" {
			continue
		}
		appendString(decoded)
	}

	// EOF emits the accumulated character tokens and closes the open element.
	result.content = content.String()
	result.end = p.pos
	result.endLine, result.endColumn = p.line, p.col
	p.recoverOpenElementsAtEOF = true
	return result, nil
}

type scriptTextState uint8

const (
	scriptDataState scriptTextState = iota
	scriptEscapedState
	scriptEscapedDashState
	scriptEscapedDashDashState
	scriptDoubleEscapedState
	scriptDoubleEscapedDashState
	scriptDoubleEscapedDashDashState
)

func isScriptDoubleEscapedState(state scriptTextState) bool {
	switch state {
	case scriptDoubleEscapedState, scriptDoubleEscapedDashState, scriptDoubleEscapedDashDashState:
		return true
	default:
		return false
	}
}

// parseScriptDataContent implements the observable script-data escaped and
// double-escaped transitions. Script execution is outside this parser's scope;
// this scanner only determines character data and the appropriate closing tag.
func (p *HTMLParser) parseScriptDataContent() (textStateContent, error) {
	result := textStateContent{start: p.pos, startLine: p.line, startColumn: p.col}
	var content strings.Builder
	state := scriptDataState

	consumeThrough := func(end int) {
		for p.pos < end {
			r, size := p.peekHTMLRune()
			if size == 0 {
				break
			}
			content.WriteRune(r)
			p.advanceRune(size)
		}
	}

	for p.pos < len(p.content) {
		if !isScriptDoubleEscapedState(state) && p.isAppropriateTextStateEndTag("script") {
			contentEnd := p.pos
			contentEndLine, contentEndColumn := p.line, p.col
			_, emitted, err := p.parseClosingTag()
			if err != nil {
				return textStateContent{}, err
			}
			if emitted {
				result.content = content.String()
				result.end = contentEnd
				result.endLine, result.endColumn = contentEndLine, contentEndColumn
				return result, nil
			}
		}

		switch state {
		case scriptDataState:
			if strings.HasPrefix(p.content[p.pos:], "<!--") {
				consumeThrough(p.pos + len("<!--"))
				state = scriptEscapedDashDashState
				continue
			}
		case scriptEscapedState:
			if end, ok := p.scriptKeywordTransitionEnd("<script"); ok {
				consumeThrough(end)
				state = scriptDoubleEscapedState
				continue
			}
			if p.peek() == '-' {
				content.WriteByte('-')
				p.advance()
				state = scriptEscapedDashState
				continue
			}
		case scriptEscapedDashState:
			if p.peek() == '-' {
				content.WriteByte('-')
				p.advance()
				state = scriptEscapedDashDashState
				continue
			}
			state = scriptEscapedState
			continue
		case scriptEscapedDashDashState:
			switch p.peek() {
			case '>':
				content.WriteByte('>')
				p.advance()
				state = scriptDataState
				continue
			case '-':
				content.WriteByte('-')
				p.advance()
				continue
			default:
				state = scriptEscapedState
				continue
			}
		case scriptDoubleEscapedState:
			if end, ok := p.scriptKeywordTransitionEnd("</script"); ok {
				consumeThrough(end)
				state = scriptEscapedState
				continue
			}
			if p.peek() == '-' {
				content.WriteByte('-')
				p.advance()
				state = scriptDoubleEscapedDashState
				continue
			}
		case scriptDoubleEscapedDashState:
			if p.peek() == '-' {
				content.WriteByte('-')
				p.advance()
				state = scriptDoubleEscapedDashDashState
				continue
			}
			state = scriptDoubleEscapedState
			continue
		case scriptDoubleEscapedDashDashState:
			switch p.peek() {
			case '>':
				content.WriteByte('>')
				p.advance()
				state = scriptDataState
				continue
			case '-':
				content.WriteByte('-')
				p.advance()
				continue
			default:
				state = scriptDoubleEscapedState
				continue
			}
		}

		r, size := p.peekHTMLRune()
		if size == 0 {
			break
		}
		content.WriteRune(r)
		p.advanceRune(size)
	}

	result.content = content.String()
	result.end = p.pos
	result.endLine, result.endColumn = p.line, p.col
	p.recoverOpenElementsAtEOF = true
	return result, nil
}

func (p *HTMLParser) scriptKeywordTransitionEnd(keyword string) (int, bool) {
	if p.pos+len(keyword) >= len(p.content) {
		return 0, false
	}
	for index := 0; index < len(keyword); index++ {
		left := p.content[p.pos+index]
		right := keyword[index]
		if left >= 'A' && left <= 'Z' {
			left += 'a' - 'A'
		}
		if left != right {
			return 0, false
		}
	}
	delimiter := p.content[p.pos+len(keyword)]
	if !isWhitespace(delimiter) && delimiter != '/' && delimiter != '>' {
		return 0, false
	}
	return p.pos + len(keyword) + 1, true
}

func (p *HTMLParser) isAppropriateTextStateEndTag(tagName string) bool {
	prefixLength := 2 + len(tagName)
	if p.pos+prefixLength >= len(p.content) || p.content[p.pos:p.pos+2] != "</" {
		return false
	}
	for index := 0; index < len(tagName); index++ {
		character := p.content[p.pos+2+index]
		if character >= 'A' && character <= 'Z' {
			character += 'a' - 'A'
		}
		if character != tagName[index] {
			return false
		}
	}
	delimiter := p.content[p.pos+prefixLength]
	return isWhitespace(delimiter) || delimiter == '/' || delimiter == '>'
}
