package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func fixture(t *testing.T) Environment {
	t.Helper()
	home := t.TempDir()
	return Environment{Home: home, Cache: filepath.Join(home, ".cache"), Config: filepath.Join(home, ".config"), Data: filepath.Join(home, ".local/share"), UID: uint32(os.Getuid()), ProcessesKnown: true, Inventory: Inventory{Commands: map[string]string{}, Packages: map[string]bool{}, Flatpaks: map[string]bool{}}}
}

func write(t *testing.T, path, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestPlanRecognizesAppCachesAndPreservesUnknownData(t *testing.T) {
	e := fixture(t)
	cache := filepath.Join(e.Config, "discord/Cache")
	write(t, filepath.Join(cache, "blob"), "downloaded")
	cookie := filepath.Join(e.Config, "discord/Network/Cookies")
	write(t, cookie, "private")
	unknown := filepath.Join(e.Cache, "unknown-model/model.bin")
	write(t, unknown, "important")
	e.Inventory.Packages["discord"] = true
	plan := BuildPlan(context.Background(), e, 2, false, true)
	if len(plan.Candidates) != 1 || plan.Candidates[0].Path != cache || plan.Candidates[0].Status != "ready" {
		t.Fatalf("unexpected candidates: %+v", plan.Candidates)
	}
	for _, path := range []string{cookie, unknown, filepath.Join(cache, "blob")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("preview changed %s: %v", path, err)
		}
	}
}

func TestRunningBrowserSkipped(t *testing.T) {
	e := fixture(t)
	write(t, filepath.Join(e.Cache, "google-chrome/Default/Cache/file"), "cache")
	e.Processes = []Process{{Name: "chrome"}}
	p := BuildPlan(context.Background(), e, 1, true, true)
	if len(p.Candidates) != 1 || p.Candidates[0].Status != "skipped" {
		t.Fatalf("active browser was eligible: %+v", p.Candidates)
	}
	e.Processes = nil
	e.ProcessesKnown = false
	p = BuildPlan(context.Background(), e, 1, true, true)
	if p.Candidates[0].Status != "skipped" {
		t.Fatal("unknown activity must fail closed")
	}
}

func TestCacheBoundaryRejectsLinksAndMounts(t *testing.T) {
	e := fixture(t)
	personal := filepath.Join(e.Home, "Documents")
	write(t, filepath.Join(personal, "photo"), "keep")
	if err := os.MkdirAll(e.Cache, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(e.Cache, "pip")
	if err := os.Symlink(personal, link); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{link, e.Home, filepath.Dir(e.Home)} {
		root, _, err := e.OpenCache(path)
		if root != nil {
			root.Close()
		}
		if err == nil {
			t.Fatalf("unsafe cache accepted: %s", path)
		}
	}
	cache := filepath.Join(e.Cache, "npm")
	write(t, filepath.Join(cache, "blob"), "cache")
	e.Mounts = []string{filepath.Join(cache, "mounted")}
	if root, _, err := e.OpenCache(cache); err == nil {
		root.Close()
		t.Fatal("nested mount accepted")
	}
}

func TestHardLinkedCacheIsNotClaimedAsReclaimable(t *testing.T) {
	e := fixture(t)
	cache := filepath.Join(e.Cache, "pip")
	write(t, filepath.Join(cache, "blob"), "shared data")
	if err := os.Link(filepath.Join(cache, "blob"), filepath.Join(e.Home, "installed-file")); err != nil {
		t.Fatal(err)
	}
	root, _, err := e.OpenCache(cache)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	size, err := cacheSize(root, e.UID)
	if err != nil || size != 0 {
		t.Fatalf("shared data overestimated: %d %v", size, err)
	}
}

func TestTreeDoesNotFollowLinksOrCrossMountBoundaries(t *testing.T) {
	e := fixture(t)
	write(t, filepath.Join(e.Home, "normal/file"), "normal")
	outside := t.TempDir()
	write(t, filepath.Join(outside, "secret"), "secret")
	if err := os.Symlink(outside, filepath.Join(e.Home, "outside")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(e.Home, "mounted/data"), "excluded")
	e.Mounts = []string{filepath.Join(e.Home, "mounted")}
	tree := ScanTree(context.Background(), e, 2, 100)
	if tree.Files != 1 || tree.Warnings != 1 {
		t.Fatalf("unexpected tree: %+v", tree)
	}
	if !ScanTree(context.Background(), e, 2, 1).Limited {
		t.Fatal("entry bound not enforced")
	}
}

func TestRustCacheRequiresManifestAndMarkerAnywhereUnderHome(t *testing.T) {
	e := fixture(t)
	project := filepath.Join(e.Home, "work/custom-project")
	write(t, filepath.Join(project, "Cargo.toml"), "[package]\n")
	write(t, filepath.Join(project, "target/CACHEDIR.TAG"), "Signature: 8a477f597d28d172789f06886806bc55\n")
	write(t, filepath.Join(e.Home, "Documents/target/notes"), "important")
	p := BuildPlan(context.Background(), e, 1, true, false)
	if len(p.Candidates) != 1 || p.Candidates[0].Provider != "rust" {
		t.Fatalf("unexpected Rust detection: %+v", p.Candidates)
	}
}

func TestSnapParsingCannotSelectActiveOrMalformedRevisions(t *testing.T) {
	revisions := ParseSnaps("Name Version Rev Tracking Publisher Notes\nfirefox 1 22 stable moz disabled\nfirefox 2 23 stable moz -\n--bad 1 1 stable x disabled\ncore24 1 4 stable x base,disabled\n")
	if len(revisions) != 3 || !revisions[0].Disabled || revisions[1].Disabled || !revisions[2].Disabled {
		t.Fatalf("invalid revisions: %+v", revisions)
	}
}
