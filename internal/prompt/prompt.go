package prompt

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

var ansi = regexp.MustCompile(`\x1b(?:\[[0-?]*[ -/]*[@-~]|\][^\x07]*(?:\x07|\x1b\\)|[ -/]*[0-~])`)

func Normalize(s string) string {
	s = ansi.ReplaceAllString(s, "")
	var b strings.Builder
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteByte(' ')
			continue
		}
		if r == 0 || (unicode.IsControl(r) && r != ' ') {
			continue
		}
		b.WriteRune(r)
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func DisplayPrefix(s string) string {
	s = Normalize(s)
	if s == "" {
		return ""
	}
	return s + " "
}

func Width(s string) int { return runewidth.StringWidth(s) }

func Truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if Width(s) <= max {
		return s
	}
	if max == 1 {
		return "…"
	}
	limit := max - 1
	g := uniseg.NewGraphemes(s)
	var b strings.Builder
	w := 0
	for g.Next() {
		cluster := g.Str()
		cw := runewidth.StringWidth(cluster)
		if w+cw > limit {
			break
		}
		b.WriteString(cluster)
		w += cw
	}
	return b.String() + "…"
}
