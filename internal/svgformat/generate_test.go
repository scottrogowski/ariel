package svgformat

import "testing"

// This test prevents Mermaid timestamp identifiers from changing generated examples on each run.
func TestNormalizeMermaidIDs(t *testing.T) {
	input := `<svg id="mermaid-1785876143985"><style>#mermaid-1785876143985{marker-end:url(#mermaid-1785876143846_flowchart-pointEnd)}</style><marker id="mermaid-1785876143846_flowchart-pointEnd"/><text>mermaid-1785876143985 #mermaid-1785876143985</text></svg>`
	want := `<svg id="mermaid-1"><style>#mermaid-1{marker-end:url(#mermaid-2_flowchart-pointEnd)}</style><marker id="mermaid-2_flowchart-pointEnd"/><text>mermaid-1785876143985 #mermaid-1785876143985</text></svg>`

	got := normalizeMermaidIDs(input)
	if got != want {
		t.Fatalf("normalizeMermaidIDs() = %q; want %q", got, want)
	}
}

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
