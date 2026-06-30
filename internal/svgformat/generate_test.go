package svgformat

import "testing"

func TestFormatStepHeader(t *testing.T) {
	cases := []struct {
		i, total int
		label    string
		want     string
	}{
		{0, 7, "Overview", "Overview"},
		{0, 7, "", ""},
		{1, 7, "The problem", "1 of 6 — The problem"},
		{6, 7, "Last step", "6 of 6 — Last step"},
		{1, 7, "", "1 of 6"},
	}
	for _, c := range cases {
		got := formatStepHeader(c.i, c.total, c.label)
		if got != c.want {
			t.Errorf("formatStepHeader(%d, %d, %q) = %q; want %q", c.i, c.total, c.label, got, c.want)
		}
	}
}
