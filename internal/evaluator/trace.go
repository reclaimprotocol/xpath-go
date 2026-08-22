package evaluator

import (
	"fmt"
	"os"
	"sync"
)

// TraceMode controls debug logging for the XPath evaluator
type TraceMode struct {
	enabled bool
	scopes  int
	mu      sync.RWMutex
}

var globalTrace = &TraceMode{}

// EnableTrace enables trace mode for debug logging
func EnableTrace() {
	globalTrace.mu.Lock()
	defer globalTrace.mu.Unlock()
	globalTrace.enabled = true
}

// DisableTrace disables trace mode
func DisableTrace() {
	globalTrace.mu.Lock()
	defer globalTrace.mu.Unlock()
	globalTrace.enabled = false
}

// IsTraceEnabled returns true if trace mode is enabled
func IsTraceEnabled() bool {
	globalTrace.mu.RLock()
	defer globalTrace.mu.RUnlock()
	return globalTrace.enabled || globalTrace.scopes > 0
}

// BeginTrace enables trace output for one evaluation and returns an idempotent
// release function. It deliberately composes with the public EnableTrace /
// DisableTrace switch: disabling a manual trace does not silence a concurrent
// scoped debug evaluation, and releasing a scope does not disable manual
// tracing.
func BeginTrace() func() {
	globalTrace.mu.Lock()
	globalTrace.scopes++
	globalTrace.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			globalTrace.mu.Lock()
			globalTrace.scopes--
			globalTrace.mu.Unlock()
		})
	}
}

// Trace logs a message if trace mode is enabled
func Trace(format string, args ...any) {
	if IsTraceEnabled() {
		fmt.Fprintf(os.Stderr, "[XPATH-TRACE] "+format+"\n", args...)
	}
}

// TraceEvaluation logs evaluation details
func TraceEvaluation(stage, expr, nodeInfo string, result any) {
	if IsTraceEnabled() {
		fmt.Fprintf(os.Stderr, "[XPATH-TRACE] %s: expr='%s', node='%s', result=%v\n",
			stage, expr, nodeInfo, result)
	}
}

// TraceCondition logs condition evaluation details
func TraceCondition(condition, nodeText string, result bool) {
	if IsTraceEnabled() {
		fmt.Fprintf(os.Stderr, "[XPATH-TRACE] CONDITION: '%s' on node='%s' -> %v\n",
			condition, nodeText, result)
	}
}

// TraceBooleanOp logs boolean operation details
func TraceBooleanOp(op, left, right string, leftResult, rightResult, finalResult bool) {
	if IsTraceEnabled() {
		fmt.Fprintf(os.Stderr, "[XPATH-TRACE] BOOLEAN: '%s' %s '%s' -> %v %s %v = %v\n",
			left, op, right, leftResult, op, rightResult, finalResult)
	}
}
