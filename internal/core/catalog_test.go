package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadAndClusterCachesPreserveToolsSourcesAndCredentials(t *testing.T) {
	e := fixture(t)
	caches := []string{".cargo/registry/cache/crates/example.crate", ".kube/cache/discovery/server.json", ".cache/helm/repository/example-index.yaml"}
	important := []string{".cargo/bin/my-tool", ".cargo/credentials.toml", ".cargo/registry/src/example/lib.rs", ".cargo/registry/index/config.json", ".cargo/git/checkouts/source.rs", ".rustup/toolchains/stable/bin/rustc", ".kube/config", ".kube/client.key", ".config/helm/repositories.yaml", ".config/helm/registry/config.json", ".local/share/helm/plugins/my-plugin", ".cache/unrecognized/model"}
	for _, path := range append(append([]string{}, caches...), important...) {
		write(t, filepath.Join(e.Home, path), "preserve")
	}
	p := BuildPlan(context.Background(), e, 1, true, true)
	if len(p.Candidates) != 3 {
		t.Fatalf("unexpected candidates: %+v", p.Candidates)
	}
	for _, c := range p.Candidates {
		if r := CleanCandidate(context.Background(), e, c, unchanged); r.Status != "cleaned" {
			t.Fatalf("%+v", r)
		}
	}
	for _, path := range caches {
		if _, err := os.Stat(filepath.Join(e.Home, path)); !os.IsNotExist(err) {
			t.Fatal("cache retained", path, err)
		}
	}
	for _, path := range important {
		if data, err := os.ReadFile(filepath.Join(e.Home, path)); err != nil || string(data) != "preserve" {
			t.Fatal("important file changed", path, err)
		}
	}
}

func TestNewProvidersSkipActiveToolsAndSymlinkedCacheRoots(t *testing.T) {
	for _, tc := range []struct{ path, process string }{
		{".cargo/registry/cache", "cargo"},
		{".kube/cache", "kubectl"},
		{".cache/helm/repository", "helm"},
	} {
		t.Run(tc.process, func(t *testing.T) {
			e := fixture(t)
			path := filepath.Join(e.Home, tc.path)
			write(t, filepath.Join(path, "cache"), "keep")
			e.Processes = []Process{{Name: tc.process}}
			p := BuildPlan(context.Background(), e, 1, true, true)
			if len(p.Candidates) != 1 || p.Candidates[0].Status != "skipped" {
				t.Fatal("active cache selected", p.Candidates)
			}
			if err := os.Rename(path, path+"-personal"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(path+"-personal", path); err != nil {
				t.Fatal(err)
			}
			e.Processes = nil
			p = BuildPlan(context.Background(), e, 1, true, true)
			if len(p.Candidates) != 1 || p.Candidates[0].Status != "skipped" {
				t.Fatal("symlinked cache selected", p.Candidates)
			}
		})
	}
}
