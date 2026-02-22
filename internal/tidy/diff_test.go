package tidy

import (
	"regexp"
	"strings"
	"testing"
)

func TestRenderUnifiedDiff_NoChange(t *testing.T) {
	got := RenderUnifiedDiff("data/test.yaml", []byte("a: 1\n"), []byte("a: 1\n"))
	if got != "" {
		t.Fatalf("expected empty diff, got:\n%s", got)
	}
}

func TestRenderUnifiedDiff_HeadersAndLineNumbers(t *testing.T) {
	got := RenderUnifiedDiff("data/test.yaml", []byte("b: 2\na: 1\n"), []byte("a: 1\nb: 2\n"))

	if !strings.Contains(got, "diff --git a/data/test.yaml b/data/test.yaml") {
		t.Fatalf("missing diff header:\n%s", got)
	}
	if !strings.Contains(got, "@@ -1,2 +1,2 @@") {
		t.Fatalf("missing hunk header:\n%s", got)
	}
	if !strings.Contains(got, "| -b: 2") || !strings.Contains(got, "| +b: 2") {
		t.Fatalf("missing changed lines:\n%s", got)
	}

	lines := strings.Split(got, "\n")
	foundDeleteWithLine := false
	foundInsertWithLine := false
	hasDigit := regexp.MustCompile(`\d`)
	for _, line := range lines {
		if strings.Contains(line, "| -b: 2") && hasDigit.MatchString(strings.Split(line, "|")[0]) {
			foundDeleteWithLine = true
		}
		if strings.Contains(line, "| +b: 2") && hasDigit.MatchString(strings.Split(line, "|")[0]) {
			foundInsertWithLine = true
		}
	}
	if !foundDeleteWithLine || !foundInsertWithLine {
		t.Fatalf("expected line-numbered add/remove lines in diff:\n%s", got)
	}
}

func TestRenderColorUnifiedDiff_UsesANSI(t *testing.T) {
	got := RenderColorUnifiedDiff("data/test.yaml", []byte("a: 1\n"), []byte("b: 1\n"))
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected ANSI escape codes in colored diff:\n%s", got)
	}
}
