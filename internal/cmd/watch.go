package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/scottrogowski/ariel/internal/dsl"
	"github.com/scottrogowski/ariel/internal/renderer"
	"github.com/scottrogowski/ariel/internal/theme"
	"github.com/spf13/cobra"
)

var watchPort int
var watchTheme string

var watchCmd = &cobra.Command{
	Use:   "watch <file.ariel.yaml>",
	Short: watchShort,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		name := filepath.Base(path)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "%s: file not found\n", name)
			os.Exit(2)
		}

		mode, err := theme.ParseMode(watchTheme)
		if err != nil {
			fmt.Fprintf(os.Stderr, "watch: %v\n", err)
			os.Exit(1)
		}

		// Initial load — print errors but don't refuse to start.
		initialHTML := loadForWatch(path, name, watchPort, mode)

		srv := renderer.NewWatchServer(path, watchPort, initialHTML, mode)

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return fmt.Errorf("watch: failed to create watcher: %w", err)
		}
		defer watcher.Close()

		if err := watcher.Add(path); err != nil {
			return fmt.Errorf("watch: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			cancel()
		}()

		go watchLoop(ctx, watcher, path, name, srv)

		url := fmt.Sprintf("http://localhost:%d", watchPort)
		fmt.Printf("watching %s\nserving at %s\n", name, url)
		openBrowser(url)

		if err := srv.Start(ctx); err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				fmt.Fprintf(os.Stderr, "watch: %v\n", err)
				os.Exit(1)
			}
		}
		return nil
	},
}

func init() {
	watchCmd.Flags().IntVarP(&watchPort, "port", "p", 2313, watchFlagPortHelp)
	watchCmd.Flags().StringVar(&watchTheme, "theme", string(theme.ModeAuto), generateFlagThemeHelp)
	rootCmd.AddCommand(watchCmd)
}

// watchLoop debounces filesystem write/create events and calls handleFileChange
// on each stable save. Exits when ctx is cancelled or the watcher closes.
func watchLoop(ctx context.Context, watcher *fsnotify.Watcher, path, name string, srv *renderer.WatchServer) {
	var debounce <-chan time.Time
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				debounce = time.After(80 * time.Millisecond)
			}
		case <-debounce:
			debounce = nil
			handleFileChange(path, name, srv)
		case err := <-watcher.Errors:
			fmt.Fprintf(os.Stderr, "watch error: %v\n", err)
		case <-ctx.Done():
			return
		}
	}
}

// handleFileChange re-parses and fully verifies the file on each save, broadcasting
// the updated HTML to connected clients or an error overlay on failure.
func handleFileChange(path, name string, srv *renderer.WatchServer) {
	w, issues, err := dsl.ParseFile(path)
	if err != nil {
		msg := fmt.Sprintf("%s: %v", name, err)
		fmt.Fprintln(os.Stderr, msg)
		srv.BroadcastError(msg)
		return
	}
	if len(issues) > 0 {
		printIssues(name, issues)
		srv.BroadcastError(issueSummary(name, issues))
		return
	}

	issues = verifyWalkthrough(w)
	if hasErrors(issues) {
		printIssues(name, issues)
		srv.BroadcastError(issueSummary(name, issues))
		return
	}
	if len(issues) > 0 {
		printIssues(name, issues)
	}

	srv.UpdateContent(w)
	fmt.Printf("reloaded %s\n", name)
}

// loadForWatch parses and renders the file for the initial watch page load.
// On error it returns an error-state HTML page so the browser shows something.
func loadForWatch(path, name string, port int, mode theme.Mode) string {
	w, issues, err := dsl.ParseFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
		return errorHTML(fmt.Sprintf("%s: %v", name, err))
	}
	if len(issues) > 0 {
		printIssues(name, issues)
		return errorHTML(issueSummary(name, issues))
	}

	issues = verifyWalkthrough(w)
	if hasErrors(issues) {
		printIssues(name, issues)
		return errorHTML(issueSummary(name, issues))
	}
	if len(issues) > 0 {
		printIssues(name, issues)
	}

	html, rerr := renderer.RenderWatch(w, port, mode)
	if rerr != nil {
		fmt.Fprintf(os.Stderr, "%s: render error: %v\n", name, rerr)
		return errorHTML(fmt.Sprintf("render error: %v", rerr))
	}
	return html
}

// issueSummary formats a concise error count message for display in the browser error overlay.
func issueSummary(name string, issues []dsl.Issue) string {
	errCount := 0
	for _, i := range issues {
		if i.Severity == dsl.SeverityError {
			errCount++
		}
	}
	return fmt.Sprintf("%s: %d error(s) — fix the file and save to reload", name, errCount)
}

// errorHTML returns a minimal dark-themed HTML page for display when the file fails
// to parse or render during watch mode.
func errorHTML(msg string) string {
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="UTF-8"><title>ariel error</title>
<style>body{background:#0f1117;color:#fca5a5;font-family:monospace;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0}pre{padding:24px;background:#1a1d27;border:1px solid #7f1d1d;border-radius:8px;max-width:80%%}</style>
</head><body><pre>%s</pre></body></html>`, msg)
}

// openBrowser opens url in the default browser; silently does nothing on unsupported platforms.
func openBrowser(url string) {
	var browserCmd string
	switch runtime.GOOS {
	case "darwin":
		browserCmd = "open"
	case "linux":
		browserCmd = "xdg-open"
	default:
		return
	}
	_ = exec.Command(browserCmd, url).Start()
}
