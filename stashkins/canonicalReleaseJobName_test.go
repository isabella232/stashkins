package stashkins

import "testing"

func TestCanonicalReleaseJobName(t *testing.T) {
	skins := DefaultStashkins{}
	releaseJobName := skins.canonicalReleaseJobName("proj", "somelib")
	if releaseJobName != "proj-somelib-release" {
		t.Fatalf("Want proj-somelib-release but got %s\n", releaseJobName)
	}
}
