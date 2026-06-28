package tui

import (
	"strings"
	"testing"
)

func TestAsciiColumnGraphScalesColumns(t *testing.T) {
	rows := asciiColumnGraph([]float64{0, 50, 100}, 3, 4, 100)
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	// A 100%% column fills to the very top row.
	if !strings.ContainsRune(rows[0], '█') {
		t.Fatalf("expected full block in top row:\n%s", strings.Join(rows, "\n"))
	}
	// The zero column stays empty in the bottom row.
	if rune(rows[3][0]) != ' ' {
		t.Fatalf("expected empty bottom cell for zero column, got %q", string(rows[3][0]))
	}
}

func TestGraphSectionRendersHeaderAndAxis(t *testing.T) {
	lines := graphSection("CPU %", []float64{10, 90}, 40, 5, 100, formatPercentValue)
	if len(lines) != 6 {
		t.Fatalf("expected header + 5 rows, got %d:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	if !strings.Contains(lines[0], "CPU %") || !strings.Contains(lines[0], "cur 90.0%") {
		t.Fatalf("unexpected header: %q", lines[0])
	}
	if !strings.Contains(lines[1], "100.0%") {
		t.Fatalf("expected top axis label 100.0%%, got %q", lines[1])
	}
}

func TestGraphSectionNeedsTwoSamples(t *testing.T) {
	// Before enough samples arrive the block still reserves its full height so
	// the layout doesn't jump once the graph appears.
	lines := graphSection("CPU %", []float64{42}, 40, 5, 100, formatPercentValue)
	if len(lines) != 6 {
		t.Fatalf("expected header + 5 reserved rows, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "collecting") {
		t.Fatalf("expected collecting caption, got %q", lines[0])
	}
}
