package mp4

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/scottmrogowski/ariel/internal/dsl"
)

const (
	DefaultStepDuration = 2
	mp4Timeout          = 5 * time.Minute
	browserWidth        = 1280
	browserHeight       = 800
)

var mdLinkRe = regexp.MustCompile(`\[([^\]]+)\]\(https?://[^)]+\)`)

var sectionTmpl = template.Must(
	template.New("section").Delims("[[", "]]").Parse(sectionHTMLTemplate),
)

type sectionData struct {
	Title          string
	MermaidDiagram string
}

type frame struct {
	path string
}

// Generate renders a Walkthrough as an MP4 video file at outPath. Requires ffmpeg on PATH.
func Generate(w *dsl.Walkthrough, outPath string, stepDuration int) error {
	if err := checkFFmpeg(); err != nil {
		return err
	}
	tmpDir, err := os.MkdirTemp("", "ariel-mp4-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	frames, err := captureFrames(w, tmpDir)
	if err != nil {
		return fmt.Errorf("capture frames: %w", err)
	}
	if err := assemble(frames, stepDuration, outPath, tmpDir); err != nil {
		return err
	}
	warnIfLarge(outPath)
	return nil
}

// githubUploadLimit is GitHub's drag-and-drop upload limit for PR/issue comments.
const githubUploadLimit = 10 * 1024 * 1024

// warnIfLarge prints a stderr warning if the output file exceeds GitHub's drag-and-drop
// upload limit for PR descriptions and issue comments.
func warnIfLarge(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.Size() > githubUploadLimit {
		fmt.Fprintf(os.Stderr, "warning: %s is %.1fMB — GitHub's drag-and-drop upload limit for PRs and issues is 10MB\n",
			path, float64(info.Size())/(1024*1024))
	}
}

// checkFFmpeg returns an error if ffmpeg is not found on PATH.
func checkFFmpeg() error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found on PATH — install it first (e.g. brew install ffmpeg)")
	}
	return nil
}

// captureFrames iterates sections and screenshots each step into tmpDir,
// returning the ordered list of frame paths.
func captureFrames(w *dsl.Walkthrough, tmpDir string) ([]frame, error) {
	ctx, cancel := newBrowserCtx()
	defer cancel()

	sections := w.ToSections()
	var frames []frame

	for secIdx, sec := range sections {
		htmlPath := filepath.Join(tmpDir, fmt.Sprintf("section%d.html", secIdx))
		if err := os.WriteFile(htmlPath, []byte(buildSectionHTML(w.Title, sec)), 0644); err != nil {
			return nil, fmt.Errorf("section %d: write HTML: %w", secIdx, err)
		}
		if err := chromedp.Run(ctx,
			chromedp.Navigate("file://"+htmlPath),
			chromedp.WaitVisible("#ready", chromedp.ByID),
		); err != nil {
			return nil, fmt.Errorf("section %d: load: %w", secIdx, err)
		}

		for stepIdx, step := range sec.Steps {
			f, err := captureStep(ctx, sec, step, stepIdx, len(sections), tmpDir, len(frames))
			if err != nil {
				return nil, fmt.Errorf("section %d: %w", secIdx, err)
			}
			frames = append(frames, f)
		}
	}
	return frames, nil
}

// captureStep applies step state via CDP, captures a screenshot, and writes it to
// tmpDir as frame<frameIdx>.png.
func captureStep(ctx context.Context, sec dsl.Section, step dsl.Step, stepIdx, totalSections int, tmpDir string, frameIdx int) (frame, error) {
	label := stepLabel(sec, step, stepIdx, totalSections)

	hJSON, _ := json.Marshal(strSlice(step.HighlightNodes))
	fJSON, _ := json.Marshal(strSlice(step.FocusNodes))
	narration := mdLinkRe.ReplaceAllString(step.Narration, "$1")
	js := fmt.Sprintf(`applyStep(%s,%s,%q,%q)`, hJSON, fJSON, label, narration)

	if err := chromedp.Run(ctx, chromedp.Evaluate(js, nil)); err != nil {
		return frame{}, fmt.Errorf("step %d: apply: %w", stepIdx, err)
	}

	var buf []byte
	if err := chromedp.Run(ctx, chromedp.CaptureScreenshot(&buf)); err != nil {
		return frame{}, fmt.Errorf("step %d: screenshot: %w", stepIdx, err)
	}

	framePath := filepath.Join(tmpDir, fmt.Sprintf("frame%04d.png", frameIdx))
	if err := os.WriteFile(framePath, buf, 0644); err != nil {
		return frame{}, fmt.Errorf("step %d: write frame: %w", stepIdx, err)
	}
	return frame{path: framePath}, nil
}

// newBrowserCtx creates a headless Chrome context with a 5-minute timeout.
// Three cancels are composed and returned as one; they must be called inner-to-outer
// (timeout → cdp context → allocator) to avoid leaking the Chrome process.
func newBrowserCtx() (context.Context, context.CancelFunc) {
	allocCtx, allocCancel := chromedp.NewExecAllocator(
		context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("no-sandbox", true),
			chromedp.WindowSize(browserWidth, browserHeight),
		)...,
	)
	ctx, ctxCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(string, ...any) {}))
	ctx, timeoutCancel := context.WithTimeout(ctx, mp4Timeout)
	return ctx, func() {
		timeoutCancel()
		ctxCancel()
		allocCancel()
	}
}

// buildSectionHTML renders the per-section static screenshot HTML from the section template.
func buildSectionHTML(title string, sec dsl.Section) string {
	var buf bytes.Buffer
	if err := sectionTmpl.Execute(&buf, sectionData{
		Title:          title,
		MermaidDiagram: strings.TrimRight(sec.MermaidDiagram, "\n"),
	}); err != nil {
		panic(fmt.Sprintf("section HTML template: %v", err))
	}
	return buf.String()
}

// stepLabel formats the visible label shown in the narration pane of the screenshot,
// replicating the JS step-counter logic from the interactive HTML player.
func stepLabel(sec dsl.Section, step dsl.Step, stepIdx, totalSections int) string {
	if stepIdx == 0 {
		if totalSections > 1 && sec.Title != "" {
			return sec.Title
		}
		return step.Label
	}
	label := fmt.Sprintf("%d of %d", stepIdx, len(sec.Steps)-1)
	if step.Label != "" {
		label += " — " + step.Label
	}
	if totalSections > 1 && sec.Title != "" {
		label = sec.Title + " · " + label
	}
	return label
}

// strSlice returns an empty slice instead of nil, ensuring JSON encodes as [] not null.
func strSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// GenerateGIF renders a Walkthrough as an animated GIF at outPath. Requires ffmpeg on PATH.
// Uses a two-pass palette approach for accurate colours. Output is scaled to 960px wide.
func GenerateGIF(w *dsl.Walkthrough, outPath string, stepDuration int) error {
	if err := checkFFmpeg(); err != nil {
		return err
	}
	tmpDir, err := os.MkdirTemp("", "ariel-gif-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	frames, err := captureFrames(w, tmpDir)
	if err != nil {
		return fmt.Errorf("capture frames: %w", err)
	}
	if err := assembleGIF(frames, stepDuration, outPath, tmpDir); err != nil {
		return err
	}
	warnIfLarge(outPath)
	return nil
}

// assembleGIF encodes frames into an animated GIF using ffmpeg's two-pass palette pipeline.
// Pass 1 generates an optimal palette; pass 2 encodes the GIF using it.
// Scaled to 960px wide (lanczos) to keep file sizes reasonable.
func assembleGIF(frames []frame, stepDuration int, outPath, tmpDir string) error {
	if len(frames) == 0 {
		return fmt.Errorf("no frames to assemble")
	}
	inputPattern := filepath.Join(tmpDir, "frame%04d.png")
	palettePath := filepath.Join(tmpDir, "palette.png")
	fps := fmt.Sprintf("1/%d", stepDuration)
	scale := "scale=960:-1:flags=lanczos"

	pass1 := exec.Command("ffmpeg", "-y",
		"-framerate", fps,
		"-i", inputPattern,
		"-vf", scale+",palettegen=stats_mode=full",
		palettePath,
	)
	if out, err := pass1.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg palettegen failed: %w\n%s", err, out)
	}

	pass2 := exec.Command("ffmpeg", "-y",
		"-framerate", fps,
		"-i", inputPattern,
		"-i", palettePath,
		"-lavfi", "[0:v]"+scale+"[x];[x][1:v]paletteuse=dither=bayer:bayer_scale=5",
		outPath,
	)
	if out, err := pass2.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg paletteuse failed: %w\n%s", err, out)
	}
	return nil
}

// assemble encodes frames named frame0000.png, frame0001.png, … in tmpDir into an MP4.
// Uses image sequence input at 1/stepDuration fps → 25fps CFR output for broad player
// compatibility. The concat demuxer produces VFR video that many players mishandle.
func assemble(frames []frame, stepDuration int, outPath, tmpDir string) error {
	if len(frames) == 0 {
		return fmt.Errorf("no frames to assemble")
	}
	inputPattern := filepath.Join(tmpDir, "frame%04d.png")
	// scale filter: ensure even pixel dimensions (libx264 requirement).
	// -crf 26: slightly above default 23 (each ~6 units halves bitrate); good for diagram text.
	// -preset slow: better compression ratio at the cost of encoding time.
	cmd := exec.Command("ffmpeg",
		"-y",
		"-framerate", fmt.Sprintf("1/%d", stepDuration),
		"-i", inputPattern,
		"-vf", "scale=trunc(iw/2)*2:trunc(ih/2)*2",
		"-c:v", "libx264", "-r", "25",
		"-crf", "26", "-preset", "slow",
		"-pix_fmt", "yuv420p",
		"-movflags", "+faststart",
		outPath,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w\n%s", err, out)
	}
	return nil
}
