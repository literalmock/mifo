package components

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/literalmock/mifo/internal/styles"
)

// Bar renders a thin horizontal utilisation bar.
func Bar(t styles.Theme, pct float64, width int, c color.Color) string {
	if width <= 0 {
		return ""
	}
	pct = min(max(pct, 0), 100)
	filled := int(pct/100*float64(width) + 0.5)
	if pct > 0 && filled == 0 {
		filled = 1
	}
	return lipgloss.NewStyle().Foreground(c).Render(strings.Repeat(t.Sym.BarFull, filled)) +
		lipgloss.NewStyle().Foreground(t.P.Track).Render(strings.Repeat(t.Sym.BarEmpty, width-filled))
}

// Segment is part of a stacked bar.
type Segment struct {
	Value float64
	Color color.Color
}

// StackedBar renders segments proportionally to total.
func StackedBar(t styles.Theme, segs []Segment, total float64, width int) string {
	if width <= 0 || total <= 0 {
		return ""
	}
	var sb strings.Builder
	used := 0
	acc := 0.0
	for _, s := range segs {
		acc += s.Value
		end := int(acc/total*float64(width) + 0.5)
		end = min(end, width)
		if n := end - used; n > 0 {
			sb.WriteString(lipgloss.NewStyle().Foreground(s.Color).Render(strings.Repeat(t.Sym.BarFull, n)))
			used = end
		}
	}
	if used < width {
		sb.WriteString(lipgloss.NewStyle().Foreground(t.P.Track).Render(strings.Repeat(t.Sym.BarEmpty, width-used)))
	}
	return sb.String()
}

// KV renders "label     value" spanning width, with the value right-aligned.
func KV(t styles.Theme, label, value string, width int) string {
	l := t.Label.Render(label)
	gap := width - lipgloss.Width(l) - lipgloss.Width(value)
	if gap < 1 {
		return ansi.Truncate(l+" "+value, width, "…")
	}
	return l + strings.Repeat(" ", gap) + value
}

// Fit clips every line of s to width cells and pads/clips to height lines,
// guaranteeing a block never overflows its slot.
func Fit(s string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, l := range lines {
		w := lipgloss.Width(l)
		switch {
		case w > width:
			lines[i] = ansi.Truncate(l, width, "")
		case w < width:
			lines[i] = l + strings.Repeat(" ", width-w)
		}
	}
	for len(lines) < height {
		lines = append(lines, strings.Repeat(" ", width))
	}
	return strings.Join(lines, "\n")
}

// Panel draws a bordered card with a title embedded in the top border and
// optional right-aligned info. Total size is exactly width × height.
func Panel(t styles.Theme, title, info, body string, width, height int, focused bool) string {
	if width < 6 || height < 3 {
		return Fit("", width, height)
	}
	b := t.Sym.Border
	bc := t.P.Border
	if focused {
		bc = t.P.BorderHi
	}
	bs := lipgloss.NewStyle().Foreground(bc)
	innerW := width - 2
	titleS := ""
	if title != "" {
		ts := t.Title
		if focused {
			ts = ts.Foreground(t.P.Accent)
		}
		titleS = " " + ts.Render(title) + " "
	}
	infoS := ""
	if info != "" {
		infoS = " " + info + " "
	}
	fill := innerW - 1 - lipgloss.Width(titleS) - lipgloss.Width(infoS) - 1
	if fill < 0 {
		infoS = ""
		fill = innerW - 1 - lipgloss.Width(titleS) - 1
		if fill < 0 {
			titleS = ansi.Truncate(titleS, innerW-2, "…")
			fill = innerW - 1 - lipgloss.Width(titleS) - 1
			fill = max(fill, 0)
		}
	}
	top := bs.Render(b.TopLeft+b.Top) + titleS + bs.Render(strings.Repeat(b.Top, fill)) + infoS + bs.Render(b.Top+b.TopRight)

	contentW := innerW - 2 // one space padding each side
	contentH := height - 2
	inner := Fit(body, contentW, contentH)
	lines := strings.Split(inner, "\n")
	var sb strings.Builder
	sb.WriteString(top)
	l, r := bs.Render(b.Left), bs.Render(b.Right)
	for _, line := range lines {
		sb.WriteString("\n" + l + " " + line + " " + r)
	}
	sb.WriteString("\n" + bs.Render(b.BottomLeft+strings.Repeat(b.Bottom, innerW)+b.BottomRight))
	return sb.String()
}

// PanelContentSize returns the usable body size for a Panel of the given size.
func PanelContentSize(width, height int) (int, int) {
	return max(width-4, 0), max(height-2, 0)
}

// Unavailable renders a polished "not available" state.
func Unavailable(t styles.Theme, title, reason string, width int) string {
	head := t.Muted.Render(t.Sym.Dot+" ") + t.Value.Render(title)
	body := lipgloss.NewStyle().Foreground(t.P.Faint).Width(max(width, 10)).Render(reason)
	return head + "\n" + body
}

// Columns lays blocks out side by side with a gap.
func Columns(gap int, blocks ...string) string {
	if len(blocks) == 0 {
		return ""
	}
	sp := strings.Repeat(" ", gap)
	parts := make([]string, 0, len(blocks)*2)
	for i, b := range blocks {
		if i > 0 {
			parts = append(parts, sp)
		}
		parts = append(parts, b)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// Split divides total into n parts separated by gap, distributing remainder.
func Split(total, n, gap int) []int {
	if n <= 0 {
		return nil
	}
	avail := total - gap*(n-1)
	out := make([]int, n)
	for i := range out {
		out[i] = avail / n
		if i < avail%n {
			out[i]++
		}
	}
	return out
}

// Table renders a simple static table. widths are column widths; align
// marks right-aligned columns.
func Table(t styles.Theme, headers []string, rows [][]string, widths []int, right []bool, selected int) string {
	var sb strings.Builder
	cell := func(s string, w int, r bool) string {
		s = ansi.Truncate(s, w, "…")
		pad := w - lipgloss.Width(s)
		if pad <= 0 {
			return s
		}
		if r {
			return strings.Repeat(" ", pad) + s
		}
		return s + strings.Repeat(" ", pad)
	}
	var hdr []string
	for i, h := range headers {
		hdr = append(hdr, cell(h, widths[i], right[i]))
	}
	sb.WriteString(t.TableHead.Render(strings.Join(hdr, " ")))
	for ri, row := range rows {
		var cells []string
		for i := range headers {
			v := ""
			if i < len(row) {
				v = row[i]
			}
			cells = append(cells, cell(v, widths[i], right[i]))
		}
		line := strings.Join(cells, " ")
		if ri == selected {
			line = t.Selected.Render(ansi.Strip(line))
		}
		sb.WriteString("\n" + line)
	}
	return sb.String()
}
