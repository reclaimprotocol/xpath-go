# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- HTML parsing now rejects malformed elements instead of silently returning a partial document.

### Tests
- Added unit coverage for malformed HTML and binary input rejection.

## [1.3.1] - 2026-03-27

### Fixed
- Improved HTML parser resilience by gracefully ignoring elements containing invalid child markup (such as unclosed/malformed tags) to prevent full document parsing failure, making xpath evaluation more forgiving.
- Added specific unit test case for parsed invalid HTML.
