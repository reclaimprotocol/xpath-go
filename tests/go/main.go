package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/reclaimprotocol/xpath-go"
)

type TestResult struct {
	Results []xpath.Result `json:"results"`
	Count   int            `json:"count"`
	Error   string         `json:"error,omitempty"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <html_file> <xpath_file> [--trace] [--contents-only]\n", os.Args[0])
		os.Exit(1)
	}

	htmlFile := os.Args[1]
	xpathFile := os.Args[2]

	// Check for options
	traceMode := false
	contentsOnly := false

	for i := 3; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--trace":
			traceMode = true
		case "--contents-only":
			contentsOnly = true
		}
	}

	if traceMode {
		xpath.EnableTrace()
		defer xpath.DisableTrace()
		fmt.Fprintf(os.Stderr, "[TRACE-ENABLED] XPath trace mode enabled\n")
	}

	if contentsOnly {
		fmt.Fprintf(os.Stderr, "[CONTENTS-ONLY] Content-only extraction mode enabled\n")
	}

	// Read HTML content
	htmlContent, err := os.ReadFile(htmlFile)
	if err != nil {
		result := TestResult{
			Results: []xpath.Result{},
			Count:   0,
			Error:   fmt.Sprintf("Failed to read HTML file: %v", err),
		}
		outputJSON(result)
		return
	}

	// Read XPath expression
	xpathContent, err := os.ReadFile(xpathFile)
	if err != nil {
		result := TestResult{
			Results: []xpath.Result{},
			Count:   0,
			Error:   fmt.Sprintf("Failed to read XPath file: %v", err),
		}
		outputJSON(result)
		return
	}

	xpathExpr := string(xpathContent)
	html := string(htmlContent)

	// Execute XPath query with panic recovery
	var results []xpath.Result
	var xpathErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				xpathErr = fmt.Errorf("panic during XPath execution: %v", r)
			}
		}()
		results, xpathErr = xpath.QueryWithOptions(xpathExpr, html, xpath.Options{
			IncludeLocation: true,
			OutputFormat:    "nodes",
			ContentsOnly:    contentsOnly,
		})
	}()

	if xpathErr != nil {
		err = xpathErr
	}

	if err != nil {
		result := TestResult{
			Results: []xpath.Result{},
			Count:   0,
			Error:   fmt.Sprintf("XPath execution error: %v", err),
		}
		outputJSON(result)
		return
	}

	// Ensure results is never nil for JSON marshaling
	if results == nil {
		results = []xpath.Result{}
	}

	// Return successful result
	result := TestResult{
		Results: results,
		Count:   len(results),
	}
	outputJSON(result)
}

func outputJSON(result TestResult) {
	jsonOutput, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON marshal error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonOutput))
}
