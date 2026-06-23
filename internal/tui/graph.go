package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/pzep1/lazycont/internal/containercli"
)

// graphBlocks are the eighth-height block runes used to draw column graphs with
// sub-row precision.
var graphBlocks = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// asciiColumnGraph renders values as a column/area graph of the given width and
// height (one column per sample, most recent samples kept). Each value is
// scaled against [0, scaleMax]; columns use eighth-block runes for smoothness.
func asciiColumnGraph(values []float64, width int, height int, scaleMax float64) []string {
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	if len(values) > width {
		values = values[len(values)-width:]
	}
	n := len(values)
	rows := make([][]rune, height)
	for r := range rows {
		rows[r] = make([]rune, n)
		for c := range rows[r] {
			rows[r][c] = ' '
		}
	}
	if scaleMax <= 0 {
		scaleMax = 1
	}
	for c := 0; c < n; c++ {
		ratio := values[c] / scaleMax
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		eighths := int(math.Round(ratio * float64(height) * 8))
		for r := 0; r < height; r++ {
			fromBottom := height - 1 - r
			cell := eighths - fromBottom*8
			switch {
			case cell >= 8:
				rows[r][c] = '█'
			case cell <= 0:
				rows[r][c] = ' '
			default:
				rows[r][c] = graphBlocks[cell]
			}
		}
	}
	out := make([]string, height)
	for r := range rows {
		out[r] = string(rows[r])
	}
	return out
}

// graphSection renders a labelled graph block: a caption with current/max
// readings, then the graph rows with a y-axis. format renders a single value
// for the caption and axis labels.
func graphSection(caption string, values []float64, width int, height int, scaleMax float64, format func(float64) string) []string {
	if len(values) < 2 {
		return []string{caption + "  (collecting samples…)"}
	}
	current := values[len(values)-1]
	observedMax := values[0]
	for _, v := range values {
		observedMax = math.Max(observedMax, v)
	}
	axisMax := scaleMax
	if axisMax <= 0 || observedMax > axisMax {
		axisMax = observedMax
	}
	if axisMax <= 0 {
		axisMax = 1
	}

	axisWidth := 7
	plotWidth := width - axisWidth - 2
	if plotWidth < 4 {
		plotWidth = 4
	}
	graph := asciiColumnGraph(values, plotWidth, height, axisMax)

	header := fmt.Sprintf("%s  cur %s  max %s", caption, format(current), format(axisMax))
	lines := make([]string, 0, height+1)
	lines = append(lines, header)
	for r, row := range graph {
		axis := strings.Repeat(" ", axisWidth)
		switch r {
		case 0:
			axis = fmt.Sprintf("%*s", axisWidth, format(axisMax))
		case height - 1:
			axis = fmt.Sprintf("%*s", axisWidth, format(0))
		}
		lines = append(lines, axis+" │"+row)
	}
	return lines
}

func formatPercentValue(v float64) string {
	return fmt.Sprintf("%.1f%%", v)
}

func formatBytesValue(v float64) string {
	if v <= 0 {
		return "0 B"
	}
	return containercli.FormatBytes(int64(v))
}
