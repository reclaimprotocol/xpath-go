// Package xpath evaluates XPath expressions against browser-recovered HTML
// documents.
//
// Query is the convenient one-shot API. Compile returns an immutable expression
// that can be reused for concurrent evaluations with XPath.Evaluate. Results
// include node metadata and, when requested, source-byte locations.
package xpath
