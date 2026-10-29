package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func unchanged(e Environment) (Environment, error) { return e, nil }

func candidate(t *testing.T, e Environment, provider string) Candidate {
	t.Helper()
	for _, c := range BuildPlan(context.Background(), e, 1, true, false).Candidates {
		if c.Provider == provider {
			return c
		}
	}
	t.Fatal("missing fixture candidate", provider)
	return Candidate{}
}

func TestCleanOnlyApprovedCacheAndPreserveProfilesLinksAndSharedFiles(t *testing.T) {
	e := fixture(t)
	cache := filepath.Join(e.Config, "discord/Cache")
	write(t, filepath.Join(cache, "blob"), "cache")
	profile := filepath.Join(e.Config, "discord/Network/Cookies")
	write(t, profile, "keep")
	personal := filepath.Join(e.Home, "Documents/photo")
	write(t, personal, "keep")
	if err := os.Symlink(filepath.Dir(personal), filepath.Join(cache, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(personal, filepath.Join(cache, "hardlink")); err != nil {
		t.Fatal(err)
	}
	result := CleanCandidate(context.Background(), e, candidate(t, e, "discord"), unchanged)
	if result.Status != "cleaned" {
		t.Fatalf("cleanup failed: %+v", result)
	}
	entries, err := os.ReadDir(cache)
	if err != nil || len(entries) != 0 {
		t.Fatal("cache not emptied", err)
	}
	for _, path := range []string{personal, profile} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "keep" {
			t.Fatal("personal data changed", path, err)
		}
	}
}

func TestChangedDirectoryAndParentSymlinkAreSkipped(t *testing.T) {
	for _, symlink := range []bool{false, true} {
		t.Run(map[bool]string{false: "replacement", true: "symlink"}[symlink], func(t *testing.T) {
			e := fixture(t)
			cache := filepath.Join(e.Cache, "pip")
			write(t, filepath.Join(cache, "blob"), "original")
			c := candidate(t, e, "pip")
			if err := os.Rename(cache, cache+"-original"); err != nil {
				t.Fatal(err)
			}
			if symlink {
				if err := os.Symlink(cache+"-original", cache); err != nil {
					t.Fatal(err)
				}
			} else {
				write(t, filepath.Join(cache, "important"), "new")
			}
			result := CleanCandidate(context.Background(), e, c, unchanged)
			if result.Status != "skipped" {
				t.Fatalf("changed cache cleaned: %+v", result)
			}
			if _, err := os.Stat(filepath.Join(cache+"-original", "blob")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNewActivityOrMountBlocksCleanup(t *testing.T) {
	for _, mount := range []bool{false, true} {
		t.Run(map[bool]string{false: "activity", true: "mount"}[mount], func(t *testing.T) {
			e := fixture(t)
			cache := filepath.Join(e.Cache, "pip")
			write(t, filepath.Join(cache, "blob"), "keep")
			c := candidate(t, e, "pip")
			refresh := func(e Environment) (Environment, error) {
				if mount {
					e.Mounts = []string{filepath.Join(cache, "nested")}
				} else {
					e.Processes = []Process{{Name: "pip"}}
				}
				return e, nil
			}
			if r := CleanCandidate(context.Background(), e, c, refresh); r.Status != "skipped" {
				t.Fatalf("unsafe cleanup: %+v", r)
			}
			if _, err := os.Stat(filepath.Join(cache, "blob")); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestForgedActionCannotDeleteUnknownFiles(t *testing.T) {
	e := fixture(t)
	path := filepath.Join(e.Home, "Documents")
	write(t, filepath.Join(path, "keep"), "keep")
	c := Candidate{ID: "forged", Provider: "pip", Path: path, Action: "directory", Scope: "user", Status: "ready"}
	if r := CleanCandidate(context.Background(), e, c, unchanged); r.Status != "skipped" {
		t.Fatalf("forged action accepted: %+v", r)
	}
}

func TestReadonlyGoModulesCleanWithoutCompilerExecution(t *testing.T) {
	e := fixture(t)
	e.Inventory.Commands["go"] = "/unused/go"
	dir := filepath.Join(e.Home, "go/pkg/mod/example")
	write(t, filepath.Join(dir, "module.go"), "cache")
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatal(err)
	}
	r := CleanCandidate(context.Background(), e, candidate(t, e, "go-modules"), unchanged)
	if r.Status != "cleaned" {
		t.Fatalf("readonly cache failed: %+v", r)
	}
}

type fakeRunner struct {
	calls []string
	reply func(string, []string) (string, error)
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	if f.reply != nil {
		return f.reply(name, args)
	}
	return "", nil
}

func TestRemoteDockerCannotBePrunedAndLocalEndpointIsPinned(t *testing.T) {
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_CONTEXT", "")
	for _, remote := range []bool{true, false} {
		e := fixture(t)
		e.Inventory.Commands["docker"] = "/usr/bin/docker"
		fake := &fakeRunner{reply: func(name string, args []string) (string, error) {
			if name == "/usr/bin/env" {
				return "cleaned", nil
			}
			if remote {
				return `"ssh://remote"`, nil
			}
			return `"unix:///run/docker.sock"`, nil
		}}
		e.Runner = fake
		c := Candidate{ID: "docker-build", Action: "docker-build", Scope: "user", Status: "ready"}
		result := CleanCandidate(context.Background(), e, c, unchanged)
		if remote && result.Status != "skipped" {
			t.Fatal("remote daemon cleaned")
		}
		if !remote && (result.Status != "cleaned" || !strings.Contains(fake.calls[len(fake.calls)-1], "--host unix:///run/docker.sock")) {
			t.Fatalf("local endpoint not pinned: %+v %v", result, fake.calls)
		}
	}
}

func TestKernelGuardProtectsRunningNewestAndNumericOrdering(t *testing.T) {
	packages := "ii linux-image-7.0.0-9-generic\nii linux-image-7.0.0-100-generic\nii linux-image-7.0.0-38-generic\n"
	for _, name := range []string{"linux-image-7.0.0-38-generic", "linux-headers-7.0.0-100", "linux-modules-extra-7.0.0-100-generic:amd64"} {
		if KernelGuard("Remv "+name, packages, "7.0.0-38-generic") == nil {
			t.Fatal("protected kernel allowed", name)
		}
	}
	if err := KernelGuard("Remv linux-image-7.0.0-9-generic", packages, "7.0.0-38-generic"); err != nil {
		t.Fatal(err)
	}
	if KernelGuard("Remv anything", "", "unknown") == nil {
		t.Fatal("unverifiable kernel state accepted")
	}
}

func TestSystemHelperRejectsUnknownActionsAndUnsafeAPTPlan(t *testing.T) {
	runner := &fakeRunner{reply: func(name string, args []string) (string, error) {
		if strings.Contains(name, "dpkg-query") {
			return "ii linux-image-7.0.0-38-generic\n", nil
		}
		return "Remv linux-image-7.0.0-38-generic\n", nil
	}}
	results := RunSystem(context.Background(), []string{"/tmp/arbitrary-command", "apt-autoremove"}, runner, "7.0.0-38-generic")
	if results[0].Status != "failed" || results[1].Status != "failed" || len(runner.calls) != 2 {
		t.Fatalf("unsafe actions executed: %+v %v", results, runner.calls)
	}
}

func TestSnapRechecksRevisionBeforeRemovingAndPreservesActive(t *testing.T) {
	for _, changed := range []bool{true, false} {
		runner := &fakeRunner{reply: func(_ string, args []string) (string, error) {
			if args[0] == "remove" {
				if args[2] != "--revision=22" {
					return "", errors.New("active revision selected")
				}
				return "removed", nil
			}
			listing := "Name Version Rev Tracking Publisher Notes\nfirefox 1 22 stable moz disabled\nfirefox 2 23 stable moz -\n"
			if len(args) == 3 && changed {
				listing = "Name Version Rev Tracking Publisher Notes\nfirefox 1 22 stable moz -\n"
			}
			return listing, nil
		}}
		r := RunSystem(context.Background(), []string{"snap-disabled"}, runner, "")
		if changed && r[0].Status != "skipped" {
			t.Fatal("changed Snap revision removed")
		}
		if !changed && (r[0].Status != "cleaned" || len(runner.calls) != 3) {
			t.Fatalf("disabled revision not removed: %+v %v", r, runner.calls)
		}
	}
}

func TestLockPreventsConcurrentCleanup(t *testing.T) {
	e := fixture(t)
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	release, err := AcquireLock(e)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if release2, err := AcquireLock(e); err == nil {
		release2()
		t.Fatal("concurrent cleanup lock accepted")
	}
}

func TestRunningGeneratedBinaryIsPreservedWithoutRecognizedProcessName(t *testing.T) {
	e := fixture(t)
	project := filepath.Join(e.Home, "work/sample")
	write(t, filepath.Join(project, "Cargo.toml"), "[package]")
	write(t, filepath.Join(project, "target/CACHEDIR.TAG"), "Signature: 8a477f597d28d172789f06886806bc55")
	binary := filepath.Join(project, "target/debug/my-app")
	write(t, binary, "generated")
	c := candidate(t, e, "rust")
	refresh := func(e Environment) (Environment, error) {
		e.Processes = []Process{{Name: "my-app", Paths: []string{binary}}}
		return e, nil
	}
	if r := CleanCandidate(context.Background(), e, c, refresh); r.Status != "skipped" {
		t.Fatalf("running binary deleted: %+v", r)
	}
	if _, err := os.Stat(binary); err != nil {
		t.Fatal(err)
	}
}

func TestFreshUbuntuSystemPlanAssociatesEligibilityWithCorrectProvider(t *testing.T) {
	e := fixture(t)
	e.Inventory.Commands = map[string]string{"apt-get": "/usr/bin/apt-get", "snap": "/usr/bin/snap", "docker": "/usr/bin/docker"}
	e.Runner = &fakeRunner{reply: func(name string, args []string) (string, error) {
		if name == "/usr/bin/docker" {
			return `"unix:///run/docker.sock"`, nil
		}
		return "0 packages to remove", nil
	}}
	p := BuildPlan(context.Background(), e, 1, true, true)
	items := map[string]Candidate{}
	for _, c := range p.Candidates {
		items[c.ID] = c
	}
	if len(items) != 4 {
		t.Fatalf("fresh system candidates: %+v", items)
	}
	if items["apt-cache"].Status != "ready" || items["docker-build"].Status != "ready" {
		t.Fatalf("eligible native caches skipped: %+v", items)
	}
	if items["apt-autoremove"].Status != "skipped" || !strings.Contains(items["apt-autoremove"].Reason, "unused packages") {
		t.Fatal("APT status attached to wrong action")
	}
	if items["snap-disabled"].Status != "skipped" || !strings.Contains(items["snap-disabled"].Reason, "Snap") {
		t.Fatal("Snap status attached to wrong action")
	}
}
