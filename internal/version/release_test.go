package version

import "testing"

// Pin the intended release independently of the implementation so a source
// version bump cannot leave the default artifact or details at the old release.
func TestDefaultReleaseVersion(t *testing.T) {
	const want = "v0.9.2"
	if Version != want || Details().Version != want {
		t.Fatalf("release version=%q details=%q, want %q", Version, Details().Version, want)
	}
}
