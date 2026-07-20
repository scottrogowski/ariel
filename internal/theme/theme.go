// Package theme is the single source of truth for ariel's colors. Every renderer
// (HTML, SVG, MP4, and the SVG extraction page) derives its palette, CSS variables,
// and Mermaid configuration from a Palette here rather than hardcoding hex values.
package theme

import (
	"fmt"
	"strconv"
	"strings"
)

// Palette holds every color a walkthrough renders. Chrome tokens style the page
// frame; diagram tokens feed Mermaid and the highlight/focus/dim states.
type Palette struct {
	// MermaidBase is the built-in Mermaid theme the diagram config extends
	// ("dark" or "base"); themeVariables below override its colors.
	MermaidBase string

	// Chrome.
	Bg           string
	Surface      string
	Border       string
	BorderSubtle string
	Text         string
	Muted        string
	Accent       string
	AccentHover  string
	AccentBright string
	LinkHover    string
	Success      string
	DotHover     string
	NarrationBg  string
	OnAccent     string

	// Diagram.
	NodeFill      string
	NodeBorder    string
	Edge          string
	ArrowHead     string
	HighlightFill string
	FocusFill     string
	DimFill       string
}

// Mode selects how a walkthrough resolves its palette. Renderers that bake pixels
// at generate time (SVG, MP4) treat auto as dark; only HTML follows the OS at
// view time via prefers-color-scheme.
type Mode string

const (
	ModeAuto  Mode = "auto"
	ModeDark  Mode = "dark"
	ModeLight Mode = "light"
)

// ParseMode validates a --theme flag value.
func ParseMode(s string) (Mode, error) {
	switch Mode(s) {
	case ModeAuto, ModeDark, ModeLight:
		return Mode(s), nil
	}
	return "", fmt.Errorf("invalid theme %q (want auto, dark, or light)", s)
}

// Baked returns the single palette to bake into a static artifact. Auto and dark
// both bake Dark; only light bakes Light.
func (m Mode) Baked() Palette {
	if m == ModeLight {
		return Light
	}
	return Dark
}

// Dark is ariel's default palette.
var Dark = Palette{
	MermaidBase:   "dark",
	Bg:            "#0f1117",
	Surface:       "#1a1d27",
	Border:        "#2a2d3a",
	BorderSubtle:  "#1e2130",
	Text:          "#e8eaf0",
	Muted:         "#6b7280",
	Accent:        "#5b8dee",
	AccentHover:   "#4a7de0",
	AccentBright:  "#7da9f0",
	LinkHover:     "#7aaaf5",
	Success:       "#4ecdc4",
	DotHover:      "#4a5a7a",
	NarrationBg:   "#141720",
	OnAccent:      "white",
	NodeFill:      "#1a2744",
	NodeBorder:    "#2a3a5a",
	Edge:          "#4a5568",
	ArrowHead:     "lightgrey",
	HighlightFill: "#1e3a6e",
	FocusFill:     "#1a4a7a",
	DimFill:       "#111520",
}

// Light is the light-mode palette. Mermaid extends the "base" theme (not "dark")
// so unspecified diagram elements resolve to light defaults.
var Light = Palette{
	MermaidBase:   "base",
	Bg:            "#ffffff",
	Surface:       "#f5f6f8",
	Border:        "#d8dbe2",
	BorderSubtle:  "#e5e7ec",
	Text:          "#1a1d27",
	Muted:         "#5a6270",
	Accent:        "#3b6fd6",
	AccentHover:   "#2f5cbf",
	AccentBright:  "#5b8dee",
	LinkHover:     "#2f5cbf",
	Success:       "#1a9d94",
	DotHover:      "#a8b0c0",
	NarrationBg:   "#f0f2f5",
	OnAccent:      "white",
	NodeFill:      "#eaf0fb",
	NodeBorder:    "#b9cdf0",
	Edge:          "#8a93a6",
	ArrowHead:     "#8a93a6",
	HighlightFill: "#cfe0fb",
	FocusFill:     "#b3d3f5",
	DimFill:       "#f0f2f5",
}

// cssVars returns the ordered CSS custom properties for a :root block. Glows and
// the CTA overlay are derived from base colors so they track the theme.
func (p Palette) cssVars() [][2]string {
	return [][2]string{
		{"--bg", p.Bg},
		{"--border", p.Border},
		{"--border-subtle", p.BorderSubtle},
		{"--text", p.Text},
		{"--muted", p.Muted},
		{"--accent", p.Accent},
		{"--accent-hover", p.AccentHover},
		{"--accent-bright", p.AccentBright},
		{"--link-hover", p.LinkHover},
		{"--success", p.Success},
		{"--dot-hover", p.DotHover},
		{"--narration-bg", p.NarrationBg},
		{"--on-accent", p.OnAccent},
		{"--node-fill", p.NodeFill},
		{"--highlight-fill", p.HighlightFill},
		{"--focus-fill", p.FocusFill},
		{"--dim-fill", p.DimFill},
		{"--accent-glow", rgba(p.Accent, "0.3")},
		{"--success-glow", rgba(p.Success, "0.3")},
		{"--focus-glow", rgba(p.Success, "0.4")},
		{"--cta-overlay", rgba(p.Bg, "0.45")},
	}
}

// RootBlock renders the CSS :root declaration block (indented for readability).
func (p Palette) RootBlock() string {
	var b strings.Builder
	b.WriteString(":root {\n")
	for _, kv := range p.cssVars() {
		fmt.Fprintf(&b, "    %s: %s;\n", kv[0], kv[1])
	}
	b.WriteString("  }")
	return b.String()
}

// HTMLRootCSS renders the :root block for HTML output. In auto mode it appends a
// prefers-color-scheme:light override so the page tracks the OS at view time;
// otherwise it bakes the single resolved palette.
func HTMLRootCSS(m Mode) string {
	if m != ModeAuto {
		return m.Baked().RootBlock()
	}
	var b strings.Builder
	b.WriteString(Dark.RootBlock())
	b.WriteString("\n\n  @media (prefers-color-scheme: light) {\n    :root {\n")
	for _, kv := range Light.cssVars() {
		fmt.Fprintf(&b, "      %s: %s;\n", kv[0], kv[1])
	}
	b.WriteString("    }\n  }")
	return b.String()
}

// HTMLMermaidConfigJS renders a JS function returning the Mermaid config. In auto
// mode the function reads prefers-color-scheme at call time, so re-invoking it
// after an OS theme change yields the matching diagram colors.
func HTMLMermaidConfigJS(m Mode) string {
	if m != ModeAuto {
		return "function arielMermaidConfig() { return " + m.Baked().mermaidConfigObject() + "; }"
	}
	return "function arielMermaidConfig() {\n" +
		"  var light = " + Light.mermaidConfigObject() + ";\n" +
		"  var dark = " + Dark.mermaidConfigObject() + ";\n" +
		"  return window.matchMedia('(prefers-color-scheme: light)').matches ? light : dark;\n" +
		"}"
}

// HTMLThemeListenerJS registers a prefers-color-scheme listener that re-renders
// the diagram on an OS theme change. Empty for baked modes (nothing to switch).
func HTMLThemeListenerJS(m Mode) string {
	if m != ModeAuto {
		return ""
	}
	return "window.matchMedia('(prefers-color-scheme: light)').addEventListener('change', reapplyTheme);"
}

// MermaidInit renders the full mermaid.initialize(...) call for this palette.
func (p Palette) MermaidInit() string {
	return "mermaid.initialize(" + p.mermaidConfigObject() + ");"
}

// mermaidConfigObject renders the config object literal passed to mermaid.initialize.
func (p Palette) mermaidConfigObject() string {
	return fmt.Sprintf(`{
  startOnLoad: false,
  theme: '%s',
  themeVariables: {
    primaryColor: '%s',
    primaryTextColor: '%s',
    primaryBorderColor: '%s',
    lineColor: '%s',
    secondaryColor: '%s',
    tertiaryColor: '%s',
    background: '%s',
    mainBkg: '%s',
    nodeBorder: '%s',
    clusterBkg: '%s',
    titleColor: '%s',
    edgeLabelBackground: '%s',
    fontFamily: 'Inter, system-ui, sans-serif'
  }
}`,
		p.MermaidBase, p.NodeFill, p.Text, p.NodeBorder, p.Edge, p.Surface, p.Surface,
		p.Bg, p.NodeFill, p.NodeBorder, p.Surface, p.Text, p.Surface)
}

// DiagramColorsJS renders a JS const holding the highlight/focus/dim colors that
// the SVG extraction page applies inline. Inline styles let each extracted SVG
// stay self-contained (and survive GitHub's SVG sanitizer).
func (p Palette) DiagramColorsJS() string {
	return fmt.Sprintf(`const ARIEL_COLORS = {focusFill:'%s',focusStroke:'%s',highlightFill:'%s',highlightStroke:'%s',dimFill:'%s',edgeStroke:'%s',arrowHead:'%s'};`,
		p.FocusFill, p.Success, p.HighlightFill, p.Accent, p.DimFill, p.Accent, p.ArrowHead)
}

// rgba converts "#rrggbb" to "rgba(r, g, b, a)". Panics on malformed input so a
// bad palette constant fails at startup rather than rendering wrong colors.
func rgba(hexColor, alpha string) string {
	h := strings.TrimPrefix(hexColor, "#")
	if len(h) != 6 {
		panic("theme: rgba expects #rrggbb, got " + hexColor)
	}
	channel := func(s string) int {
		v, err := strconv.ParseInt(s, 16, 0)
		if err != nil {
			panic("theme: invalid hex channel in " + hexColor)
		}
		return int(v)
	}
	return fmt.Sprintf("rgba(%d, %d, %d, %s)", channel(h[0:2]), channel(h[2:4]), channel(h[4:6]), alpha)
}
