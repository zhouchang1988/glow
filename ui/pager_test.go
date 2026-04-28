package ui

import "testing"

func TestFormatRawMarkdown(t *testing.T) {
	input := "# Hello\n\n- item 1\n- item 2\n"
	got := formatRawMarkdown(input, 40)

	lines := splitLines(got)
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %q", len(lines), got)
	}

	// First line should contain line number area
	if len(lines[0]) < 4 {
		t.Fatalf("line 0 too short: %q", lines[0])
	}
}

func TestFormatRawMarkdownEmpty(t *testing.T) {
	got := formatRawMarkdown("", 40)
	if got != "" {
		t.Errorf("expected empty string for empty input, got: %q", got)
	}
}

func TestFormatRawMarkdownNarrowWidth(t *testing.T) {
	input := "# This is a very long title that exceeds the width"
	got := formatRawMarkdown(input, 20)
	if got == "" {
		t.Error("expected non-empty output")
	}
}

func TestSyncScrollMath(t *testing.T) {
	// percent=0.5, totalLines=100, height=20 -> yOffset = 0.5 * 80 = 40
	percent := 0.5
	totalLines := 100
	height := 20
	yOffset := int(percent * float64(totalLines-height))
	if yOffset != 40 {
		t.Errorf("expected yOffset 40, got %d", yOffset)
	}

	// percent=0.0 -> yOffset = 0
	yOffset = int(0.0 * float64(totalLines-height))
	if yOffset != 0 {
		t.Errorf("expected yOffset 0, got %d", yOffset)
	}

	// percent=1.0 -> yOffset = 80
	yOffset = int(1.0 * float64(totalLines-height))
	if yOffset != 80 {
		t.Errorf("expected yOffset 80, got %d", yOffset)
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	result := splitByNewline(s)
	if len(result) > 0 && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}
	return result
}

func splitByNewline(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}
