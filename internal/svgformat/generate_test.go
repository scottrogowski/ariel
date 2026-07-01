package svgformat

import "testing"

func TestFormatStepHeader(t *testing.T) {
	cases := []struct {
		sectionTitle string
		secStepIdx   int
		secTotal     int
		stepLabel    string
		multiSection bool
		want         string
	}{
		// Single-section: overview step returns its label.
		{"", 0, 7, "Overview", false, "Overview"},
		{"", 0, 7, "", false, ""},
		// Single-section: content steps use "N of M — label" format.
		{"", 1, 7, "The problem", false, "1 of 6 — The problem"},
		{"", 6, 7, "Last step", false, "6 of 6 — Last step"},
		{"", 1, 7, "", false, "1 of 6"},
		// Multi-section: overview step returns section title.
		{"The pipeline", 0, 7, "Overview", true, "The pipeline"},
		// Multi-section: content steps prefix "SECTION · N of M — label".
		{"The pipeline", 1, 7, "Parsing", true, "The pipeline · 1 of 6 — Parsing"},
		{"The pipeline", 1, 7, "", true, "The pipeline · 1 of 6"},
	}
	for _, c := range cases {
		got := formatStepHeader(c.sectionTitle, c.secStepIdx, c.secTotal, c.stepLabel, c.multiSection)
		if got != c.want {
			t.Errorf("formatStepHeader(%q, %d, %d, %q, %v) = %q; want %q",
				c.sectionTitle, c.secStepIdx, c.secTotal, c.stepLabel, c.multiSection, got, c.want)
		}
	}
}
