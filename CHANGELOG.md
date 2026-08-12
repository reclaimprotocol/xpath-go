# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
