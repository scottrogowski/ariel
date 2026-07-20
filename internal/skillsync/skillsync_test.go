package skillsync

import "testing"

// Catches SKILL.md drifting from `ariel guide` after guide.txt changes without
// a `make sync-skill`.
func TestSkillInSyncWithGuide(t *testing.T) {
	if err := Check(); err != nil {
		t.Fatal(err)
	}
}
