package utils

import (
	"strings"
	"testing"
)

const benchmarkPlainText = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. "

func BenchmarkHTMLParserPlainText(b *testing.B) {
	content := "<main><p>" + strings.Repeat(benchmarkPlainText, 256) + "</p></main>"
	b.ReportAllocs()
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		document, err := NewHTMLParser().Parse(content)
		if err != nil || document == nil {
			b.Fatalf("Parse failed: %v", err)
		}
	}
}

func BenchmarkHTMLParserMalformedFormattingRecovery(b *testing.B) {
	content := "<main>"
	for i := 0; i < 96; i++ {
		content += "<b><i><a>item</b></i></a>"
	}
	content += "</main>"
	b.ReportAllocs()
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		document, err := NewHTMLParser().Parse(content)
		if err != nil || document == nil {
			b.Fatalf("Parse recovery failed: %v", err)
		}
	}
}

func BenchmarkHTMLParserTableRecovery(b *testing.B) {
	content := "<table><tbody>"
	for i := 0; i < 96; i++ {
		content += "text<tr><td>left<td>right"
	}
	content += "</tbody></table>"
	b.ReportAllocs()
	b.SetBytes(int64(len(content)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		document, err := NewHTMLParser().Parse(content)
		if err != nil || document == nil {
			b.Fatalf("Parse table recovery failed: %v", err)
		}
	}
}
