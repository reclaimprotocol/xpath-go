# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.6.0] - 2026-08-21

### Added
- Added charset-aware XPath evaluation for raw response bodies through `QueryBytes`, `QueryBytesWithOptions`, and compiled-expression byte APIs.
- Added WHATWG-compatible charset labels, including Windows-1252 aliases, Shift_JIS, UTF-16, and stateful encodings.
- XPath text and attribute matching now uses decoded Unicode while result locations continue to index the original response bytes.

### Changed
- Compiled XPath evaluations use call-scoped evaluators so compiled expressions can be shared safely across concurrent documents and options.
- Source mapping now grows with actual mapping complexity instead of preallocating from response size.
- Stateful decoder boundaries use field-specific raw offsets so adjacent node ranges exclude shift bytes while element content ranges retain all encoded bytes between tags.

### Tests
- Added raw-byte charset regressions covering ISO-8859-1/Windows-1252, Shift_JIS, UTF-16, BOM handling, malformed input, templates, stateful boundaries, and public API paths.
- Added five alternate-charset Go-versus-jsdom cases to the comprehensive comparator.
- Verified the full Go suite, race detector, vet, the charset oracle, and comprehensive compatibility coverage.

## [1.5.1] - 2026-08-19

### Fixed
- HTML parsing now ignores any end tag whose name has no matching element in scope instead of aborting with `unexpected closing tag`, matching the browser "in body" generic end-tag rule. This recovers stray closing tags inside table and template content, where the "in table" anything-else and "in template" any-other-end-tag entries route to the same ignore rule. Previously such documents failed the whole parse and broke XPath extraction.

### Tests
- Added stray end-tag recovery regressions: a stray closing tag inside table content, outside table scope across the table boundary, inside a table cell (merged surrounding text), and a full-page portal layout asserting a nested span text query recovers.
- Added jsdom comparator cases for all four scenarios (845/845 passing).
- Added adjacent-attribute recovery regressions at a reported position with long attribute prefixes.

## [1.5.0] - 2026-08-14

### Added
- Added browser-style whole-document construction with implicit `html`, `head`, and `body` nodes, retained-head routing, EOF recovery, and minimal frameset handling.
- Added HTML template content fragments with nested template modes, table integration, formatting/form isolation, and document-XPath isolation.
- Added SVG and MathML foreign-content namespaces, integration points, breakout recovery, adjusted names, and namespaced attribute metadata.
- Added HTML processing-instruction nodes and XPath `processing-instruction()` and `comment()` node tests.
- Added XPath `following::` and `preceding::` axes with DOM-order semantics and reverse-axis predicate positioning.
- Added typed XPath node-set, string, number, and boolean evaluation, including `string()`, `number()`, `boolean()`, strict coercion/comparison rules, unions in function arguments, IEEE special values, and numeric predicates.

### Changed
- Whole-document parsing now follows the browser document model: ordinary root content is placed beneath implicit HTML wrappers and empty input produces an empty document skeleton.
- XPath node-set comparisons, conversions, unions, and axis results now preserve type, identity, and recovered DOM order instead of relying on first-value string coercion.
- Public parser nodes now expose template content and foreign attribute namespace/local-name/prefix metadata.

### Fixed
- Expanded browser recovery for malformed start/end tags, paragraphs, lists, buttons, forms, formatting/adoption cases, tables, selects, comments, doctypes, character references, and incomplete or mismatched markup.
- Corrected text-state handling for RCDATA, raw text, script escapes, plaintext, and initial-newline suppression in `pre` and `listing`.
- Preserved original UTF-8 byte ranges and UTF-16-aware line/column tracking across recovered, reconstructed, foreign, and template trees.
- Fixed attribute union identity, reverse-axis conversion order, predicate context restoration, comparison precedence, strict function arity, and overflow/`NaN`/infinity behavior.

### Tests
- Expanded the browser compatibility comparator to 878/878 passing cases.
- Added an independent 18/18 typed XPath conversion oracle plus parser, evaluator, public API, reuse, recovery, and scaling regressions.
- Verified the legacy `v1.4.7` test surface, the full Go suite, race detector, vet, lint, formatting, and diff checks.

### Compatibility notes
- Scripting-enabled parsing is outside this release's compatibility claim; the public default remains scripting-disabled.
- See `docs/COMPATIBILITY.md` for the validated scope and explicitly deferred XPath/XML capabilities.

## [1.4.7] - 2026-08-13

### Fixed
- HTML parsing now recovers non-comment `<!...>` declarations as bogus comments instead of aborting with `expected comment`, matching browser tokenization.
- Bogus comments terminate at the next `>` or EOF, remain excluded from parent text content, and preserve original byte ranges for subsequent XPath results.

### Tests
- Added Chromium-verified coverage for unknown, CDATA-like, entity, empty, and EOF-terminated declarations.
- Added a public XPath regression at the reported declaration offset `14738` with exact original-source location assertions.

## [1.4.6] - 2026-08-13

### Fixed
- HTML parsing now ignores stray closing tags for void elements instead of aborting on a mismatched closing tag, matching browser recovery for markup such as `</meta>`.
- The browser-specific `</br>` exception is recovered as a `br` element.
- Recovered nodes and subsequent XPath results retain byte positions against the original, unmodified HTML input.

### Tests
- Added coverage for all supported void-element closing tags, the `</br>` exception, the reported `<head>`/`</meta>` case, and original-source XPath ranges.

## [1.4.5] - 2026-08-13

### Fixed
- HTML parsing now tolerates adjacent attributes without separating whitespace, matching browser behavior for markup such as `id="target"class="primary"`.
- Original byte positions remain tied to the unmodified input while parsing adjacent attributes.

### Tests
- Added parser and public XPath regression coverage for adjacent quoted and boolean attributes and exact original-source ranges.

## [1.4.4] - 2026-08-13

### Fixed
- Added browser-compatible implicit closing for table rows when a new row or table boundary is encountered.
- Corrected reverse-axis indexing so `preceding-sibling::*[1]` selects the nearest matching sibling.
- Preserved byte locations against the original unrepaired HTML when rows are implicitly closed.

### Tests
- Added parser and XPath regression coverage for malformed table rows, sibling positions, and original-source location ranges.

## [1.4.3] - 2026-07-27

### Fixed
- HTML parsing now rejects malformed elements, mismatched or missing closing tags, unterminated special elements, orphan closing tags, and invalid attribute syntax instead of silently returning a partial document.
- Removed malformed-input recovery paths that could loop indefinitely or attach descendants to the wrong parent.

### Tests
- Added parser and public API coverage for malformed HTML, invalid attributes, unbalanced markup, unterminated constructs, and binary input rejection.

## [1.3.1] - 2026-03-27

### Fixed
- Improved HTML parser resilience by gracefully ignoring elements containing invalid child markup (such as unclosed/malformed tags) to prevent full document parsing failure, making xpath evaluation more forgiving.
- Added specific unit test case for parsed invalid HTML.
