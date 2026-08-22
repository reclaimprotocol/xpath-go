package utils

import (
	"strings"
	"testing"
	"time"
)

func TestParseExplicitDocumentSkeletonScalesNearLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping document-skeleton scaling regression in short mode")
	}
	measure := func(count int) time.Duration {
		content := `<html><head></head>` + strings.Repeat(`<!--late--><meta>`, count) + `<body>` + strings.Repeat(`<span>x</span>`, count) + `</body></html>`
		started := time.Now()
		document, err := NewHTMLParser().Parse(content)
		elapsed := time.Since(started)
		if err != nil {
			t.Fatalf("Parse(%d phase tokens) failed: %v", count, err)
		}
		_, head, body := requireDirectSkeleton(t, document)
		if len(head.Children) != count || len(body.Children) != count {
			t.Fatalf("Parse(%d) routed head/body counts %d/%d", count, len(head.Children), len(body.Children))
		}
		return elapsed
	}
	best := func(count int) time.Duration {
		bestDuration := time.Duration(1<<63 - 1)
		for attempt := 0; attempt < 2; attempt++ {
			if elapsed := measure(count); elapsed < bestDuration {
				bestDuration = elapsed
			}
		}
		return bestDuration
	}
	_ = measure(100)
	small, large := best(1000), best(4000)
	t.Logf("parsed late-head/body document phases at 1000=%s 4000=%s", small, large)
	if large > 12*small && large-small > 50*time.Millisecond {
		t.Fatalf("explicit document phase handling scales superlinearly: 4x input took %.1fx (%s -> %s)", float64(large)/float64(small), small, large)
	}
}
