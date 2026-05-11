package tmux

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDeriveSocketLabel_DistinctDataDirsGetDistinctLabels(t *testing.T) {
	a := deriveSocketLabel("/tmp/prod-data")
	b := deriveSocketLabel("/tmp/dev-data")
	if a == b {
		t.Fatalf("expected distinct labels for distinct data dirs, both = %q", a)
	}
}

func TestDeriveSocketLabel_StableAcrossEquivalentPaths(t *testing.T) {
	abs, err := filepath.Abs("./data")
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	if deriveSocketLabel("./data") != deriveSocketLabel(abs) {
		t.Fatalf("relative and absolute forms of the same path must hash identically")
	}
}

func TestDeriveSocketLabel_HasLociPrefix(t *testing.T) {
	got := deriveSocketLabel("/tmp/anything")
	if !strings.HasPrefix(got, "lociterm-") {
		t.Fatalf("label %q missing lociterm- prefix; would collide with the user's default tmux server", got)
	}
}
