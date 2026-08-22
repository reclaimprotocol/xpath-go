package evaluator

import (
	"sync"
	"testing"
)

func TestConcurrentTraceScopesReferenceCountAndReleaseIdempotently(t *testing.T) {
	DisableTrace()
	defer DisableTrace()

	const workers = 32
	stops := make([]func(), workers)
	var ready sync.WaitGroup
	var done sync.WaitGroup
	release := make(chan struct{})
	ready.Add(workers)
	done.Add(workers)
	for index := 0; index < workers; index++ {
		go func(index int) {
			defer done.Done()
			stops[index] = BeginTrace()
			ready.Done()
			<-release
			stops[index]()
			stops[index]()
		}(index)
	}
	ready.Wait()
	if !IsTraceEnabled() {
		t.Fatal("trace disabled while active scopes remain")
	}
	close(release)
	done.Wait()
	if IsTraceEnabled() {
		t.Fatal("trace remains enabled after all scopes are released")
	}
}
