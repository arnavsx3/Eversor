package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPinnedDirectoryChild(t *testing.T) {
	if os.Getenv("EVERSOR_TEST_PINNED_CHILD") != "1" {
		return
	}
	data, err := os.ReadFile("/proc/self/fd/3/marker")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("PINNED:" + string(data))
}

func TestNativeRunnerUsesOriginalDirectoryAfterPathReplacement(t *testing.T) {
	dir := t.TempDir()
	cache := filepath.Join(dir, "cache")
	write(t, filepath.Join(cache, "marker"), "original")
	handle, err := os.Open(cache)
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	if err := os.Rename(cache, cache+"-original"); err != nil {
		t.Fatal(err)
	}
	replacement := filepath.Join(dir, "personal")
	write(t, filepath.Join(replacement, "marker"), "personal")
	if err := os.Symlink(replacement, cache); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EVERSOR_TEST_PINNED_CHILD", "1")
	output, err := (ExecRunner{}).RunPinned(context.Background(), handle, os.Args[0], "-test.run=^TestPinnedDirectoryChild$")
	if err != nil || !strings.Contains(output, "PINNED:original") || strings.Contains(output, "PINNED:personal") {
		t.Fatalf("native tool escaped the original directory: %s %v", output, err)
	}
}

func TestInstalledUVCanPruneDisposablePinnedCache(t *testing.T) {
	uv, err := exec.LookPath("uv")
	if err != nil {
		t.Skip("optional integration check requires uv")
	}
	e := fixture(t)
	e.Runner = ExecRunner{}
	e.Inventory.Commands["uv"] = uv
	write(t, filepath.Join(e.Cache, "uv/CACHEDIR.TAG"), "Signature: 8a477f597d28d172789f06886806bc55")
	personal := filepath.Join(e.Home, "Documents/important")
	write(t, personal, "preserve")
	r := CleanCandidate(context.Background(), e, candidate(t, e, "uv"), unchanged)
	if r.Status != "cleaned" {
		t.Fatalf("uv could not use the pinned cache: %+v", r)
	}
	if data, err := os.ReadFile(personal); err != nil || string(data) != "preserve" {
		t.Fatal("personal file changed", err)
	}
}
