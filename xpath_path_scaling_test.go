package xpath

import (
	"strconv"
	"strings"
	"testing"
)

// BenchmarkResultPathScaling exercises conversion at several result sizes.
// Path generation should reuse ancestor and sibling metadata; this benchmark
// is intentionally measurement-only because absolute timings vary by host.
func BenchmarkResultPathScaling(b *testing.B) {
	for _, size := range []int{64, 256, 1024} {
		b.Run(strconv.Itoa(size), func(b *testing.B) {
			var content strings.Builder
			content.Grow(size * 32)
			content.WriteString("<root>")
			for i := 0; i < size; i++ {
				content.WriteString("<item><value>payload</value></item>")
			}
			content.WriteString("</root>")
			document := content.String()
			b.ReportAllocs()
			b.SetBytes(int64(len(document)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				results, err := QueryWithOptions(`//item/value`, document, Options{OutputFormat: "paths"})
				if err != nil || len(results) != size {
					b.Fatalf("path query failed: %v (results=%d, want=%d)", err, len(results), size)
				}
			}
		})
	}
}
