package tidy

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type diffKind int

const (
	diffEqual diffKind = iota
	diffDelete
	diffInsert
)

type diffLine struct {
	kind    diffKind
	token   string
	oldLine int
	newLine int
}

// RenderUnifiedDiff renders a git-like unified diff (without color).
func RenderUnifiedDiff(path string, original, tidied []byte) string {
	return renderUnifiedDiff(path, original, tidied, false)
}

// RenderColorUnifiedDiff renders a git-like unified diff using ANSI colors.
func RenderColorUnifiedDiff(path string, original, tidied []byte) string {
	return renderUnifiedDiff(path, original, tidied, true)
}

func renderUnifiedDiff(path string, original, tidied []byte, color bool) string {
	if bytes.Equal(original, tidied) {
		return ""
	}

	oldLines := splitDiffTokens(original)
	newLines := splitDiffTokens(tidied)
	ops := lineDiff(oldLines, newLines)
	if len(ops) == 0 {
		return ""
	}
	numberDiffLines(ops)

	oldCount := 0
	newCount := 0
	for _, op := range ops {
		if op.kind != diffInsert {
			oldCount++
		}
		if op.kind != diffDelete {
			newCount++
		}
	}

	oldStart := 1
	newStart := 1
	if oldCount == 0 {
		oldStart = 0
	}
	if newCount == 0 {
		newStart = 0
	}

	width := len(strconv.Itoa(maxInt(1, maxInt(len(oldLines), len(newLines)))))

	var b strings.Builder
	writeDiffHeader(&b, path, oldStart, oldCount, newStart, newCount, color)
	for _, op := range ops {
		writeDiffLine(&b, op, width, color)
	}
	return b.String()
}

func writeDiffHeader(b *strings.Builder, path string, oldStart, oldCount, newStart, newCount int, color bool) {
	writeColoredLine(b, fmt.Sprintf("diff --git a/%s b/%s", path, path), ansiBold, color)
	writeColoredLine(b, fmt.Sprintf("--- a/%s", path), ansiRed, color)
	writeColoredLine(b, fmt.Sprintf("+++ b/%s", path), ansiGreen, color)
	writeColoredLine(
		b,
		fmt.Sprintf("@@ -%d,%d +%d,%d @@", oldStart, oldCount, newStart, newCount),
		ansiCyan,
		color,
	)
}

func writeDiffLine(b *strings.Builder, op diffLine, width int, color bool) {
	oldNum := ""
	newNum := ""
	prefix := ' '
	lineColor := ""

	switch op.kind {
	case diffEqual:
		prefix = ' '
	case diffDelete:
		prefix = '-'
		lineColor = ansiRed
	case diffInsert:
		prefix = '+'
		lineColor = ansiGreen
	}

	if op.oldLine > 0 {
		oldNum = strconv.Itoa(op.oldLine)
	}
	if op.newLine > 0 {
		newNum = strconv.Itoa(op.newLine)
	}

	text := displayToken(op.token)
	line := fmt.Sprintf("%*s %*s | %c%s\n", width, oldNum, width, newNum, prefix, text)
	if color && lineColor != "" {
		b.WriteString(lineColor)
		b.WriteString(line)
		b.WriteString(ansiReset)
		return
	}
	b.WriteString(line)
}

func writeColoredLine(b *strings.Builder, line, ansi string, color bool) {
	if color && ansi != "" {
		b.WriteString(ansi)
		b.WriteString(line)
		b.WriteString(ansiReset)
		b.WriteByte('\n')
		return
	}
	b.WriteString(line)
	b.WriteByte('\n')
}

func splitDiffTokens(data []byte) []string {
	s := string(data)
	if s == "" {
		return nil
	}
	parts := strings.SplitAfter(s, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		return parts[:len(parts)-1]
	}
	return parts
}

func displayToken(token string) string {
	token = strings.TrimSuffix(token, "\n")
	token = strings.TrimSuffix(token, "\r")
	return token
}

func numberDiffLines(ops []diffLine) {
	oldNext := 1
	newNext := 1
	for i := range ops {
		switch ops[i].kind {
		case diffEqual:
			ops[i].oldLine = oldNext
			ops[i].newLine = newNext
			oldNext++
			newNext++
		case diffDelete:
			ops[i].oldLine = oldNext
			oldNext++
		case diffInsert:
			ops[i].newLine = newNext
			newNext++
		}
	}
}

func lineDiff(oldLines, newLines []string) []diffLine {
	if len(oldLines) == 0 && len(newLines) == 0 {
		return nil
	}

	// Avoid excessive memory for large files. In the fallback we still produce a correct
	// diff, but without minimal edit grouping.
	if int64(len(oldLines))*int64(len(newLines)) > 4_000_000 {
		return fallbackLineDiff(oldLines, newLines)
	}

	n := len(oldLines)
	m := len(newLines)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}

	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if oldLines[i] == newLines[j] {
				dp[i][j] = dp[i+1][j+1] + 1
				continue
			}
			if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	ops := make([]diffLine, 0, n+m)
	i, j := 0, 0
	for i < n && j < m {
		if oldLines[i] == newLines[j] {
			ops = append(ops, diffLine{kind: diffEqual, token: oldLines[i]})
			i++
			j++
			continue
		}
		if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, diffLine{kind: diffDelete, token: oldLines[i]})
			i++
			continue
		}
		ops = append(ops, diffLine{kind: diffInsert, token: newLines[j]})
		j++
	}
	for ; i < n; i++ {
		ops = append(ops, diffLine{kind: diffDelete, token: oldLines[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, diffLine{kind: diffInsert, token: newLines[j]})
	}
	return ops
}

func fallbackLineDiff(oldLines, newLines []string) []diffLine {
	ops := make([]diffLine, 0, len(oldLines)+len(newLines))
	for _, line := range oldLines {
		ops = append(ops, diffLine{kind: diffDelete, token: line})
	}
	for _, line := range newLines {
		ops = append(ops, diffLine{kind: diffInsert, token: line})
	}
	return ops
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

const (
	ansiReset = "\x1b[0m"
	ansiBold  = "\x1b[1m"
	ansiRed   = "\x1b[31m"
	ansiGreen = "\x1b[32m"
	ansiCyan  = "\x1b[36m"
)
