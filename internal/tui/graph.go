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

// axisWidth is the fixed-width gutter that holds the y-axis labels. Keeping it
// constant pins the "│" separator (and the whole plot) to the same column no
// matter how wide the formatted axis label is.
const axisWidth = 9

// graphSection renders a labelled graph block: a caption with current/max
// readings, then the graph rows with a y-axis. format renders a single value
// for the caption and axis labels. It always returns height+1 lines so the
// block keeps a stable footprint, even before enough samples have arrived.
func graphSection(caption string, values []float64, width int, height int, scaleMax float64, format func(float64) string) []string {
	if len(values) < 2 {
		lines := make([]string, 0, height+1)
		lines = append(lines, caption+"  (collecting samples…)")
		for len(lines) < height+1 {
			lines = append(lines, "")
		}
		return lines
	}
	current := values[len(values)-1]
	observedMax := values[0]
	for _, v := range values {
		observedMax = math.Max(observedMax, v)
	}
	// Quantise the axis ceiling to a "nice" round number so it only changes
	// when the data crosses a boundary, instead of rescaling every refresh.
	axisMax := scaleMax
	if observedMax > axisMax {
		if scaleMax > 0 {
			axisMax = niceCeil(observedMax)
		} else {
			axisMax = niceByteCeil(observedMax)
		}
	}
	if axisMax <= 0 {
		axisMax = niceByteCeil(observedMax)
	}

	plotWidth := width - axisWidth - 2
	if plotWidth < 4 {
		plotWidth = 4
	}
	graph := asciiColumnGraph(values, plotWidth, height, axisMax)

	// Left-align the current/max readings in a fixed slot so the "max" label
	// keeps its column while the live value's digit count changes.
	header := fmt.Sprintf("%s  cur %-8s  max %-8s", caption, format(current), format(axisMax))
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

// niceCeil rounds value up to the next "nice" number (1, 2, 2.5 or 5 × a power
// of ten) so an axis ceiling changes rarely and predictably.
func niceCeil(value float64) float64 {
	if value <= 0 {
		return 1
	}
	exp := math.Floor(math.Log10(value))
	base := math.Pow(10, exp)
	switch f := value / base; {
	case f <= 1:
		return base
	case f <= 2:
		return 2 * base
	case f <= 2.5:
		return 2.5 * base
	case f <= 5:
		return 5 * base
	default:
		return 10 * base
	}
}

// niceByteCeil rounds a byte count up to a nice multiple of its display unit
// (KB/MB/GB…), keeping memory-graph axis labels both stable and readable.
func niceByteCeil(value float64) float64 {
	if value <= 0 {
		return 1
	}
	unit := 1.0
	for value/unit >= 1024 && unit < math.Pow(1024, 4) {
		unit *= 1024
	}
	return niceCeil(value/unit) * unit
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

// formatRateValue renders a bytes-per-second throughput for the network and
// block-IO graphs.
func formatRateValue(v float64) string {
	if v <= 0 {
		return "0 B/s"
	}
	return containercli.FormatBytes(int64(v)) + "/s"
}
