# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
