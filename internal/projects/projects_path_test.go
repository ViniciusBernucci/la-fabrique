package projects

import "testing"

func TestValidRepoPathRejectsHiddenSegments(t *testing.T) {
	for p, want := range map[string]bool{"app": true, "grupo/app": true, ".lf-worktrees/x": false, "a/.git": false, "../x": false, "/abs": false, "-x": false} {
		if got := ValidRepoPath(p); got != want {
			t.Errorf("%q: %v", p, got)
		}
	}
}
