package xpath

import "testing"

func TestParseGoVersion(t *testing.T) {
	tests := []struct {
		version    string
		major      int
		minor      int
		acceptable bool
	}{
		{version: "go1.21.0", major: 1, minor: 21, acceptable: true},
		{version: "go1.24rc2", major: 1, minor: 24, acceptable: true},
		{version: "go2.0", major: 2, minor: 0, acceptable: true},
		{version: "devel go1.25", acceptable: false},
		{version: "go1", acceptable: false},
		{version: "1.24", acceptable: false},
	}
	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			major, minor, ok := parseGoVersion(test.version)
			if major != test.major || minor != test.minor || ok != test.acceptable {
				t.Fatalf("parseGoVersion(%q) = %d, %d, %v", test.version, major, minor, ok)
			}
		})
	}
}

func TestBuildInfoIncludesReleaseMetadata(t *testing.T) {
	previousCommit, previousBuildTime := GitCommit, BuildTime
	t.Cleanup(func() { GitCommit, BuildTime = previousCommit, previousBuildTime })
	GitCommit, BuildTime = "abc123", "2026-08-22T00:00:00Z"

	info := GetBuildInfo()
	if info.GitCommit != GitCommit || info.BuildTime != BuildTime {
		t.Fatalf("release metadata missing from build info: %#v", info)
	}
}

func TestNilXPathExpressionIsEmpty(t *testing.T) {
	var compiled *XPath
	if expression := compiled.GetExpression(); expression != "" {
		t.Fatalf("nil XPath expression = %q, want empty", expression)
	}
}
