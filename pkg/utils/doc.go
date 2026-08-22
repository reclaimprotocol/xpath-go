// Package utils provides the HTML parser used by xpath-go. HTMLParser is
// reusable, but Parse replaces its per-document state; do not call Parse
// concurrently on the same parser instance. Create one parser per concurrent
// parse operation.
package utils
