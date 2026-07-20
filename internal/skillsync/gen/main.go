// Command skillsync regenerates the `ariel guide` block inside SKILL.md.
// Run via `make sync-skill`.
package main

import (
	"fmt"
	"os"

	"github.com/scottrogowski/ariel/internal/skillsync"
)

func main() {
	path, err := skillsync.Sync()
	if err != nil {
		fmt.Fprintln(os.Stderr, "skillsync:", err)
		os.Exit(1)
	}
	fmt.Println("synced ariel guide into", path)
}
