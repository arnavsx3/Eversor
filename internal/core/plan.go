package core

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func BuildPlan(ctx context.Context, e Environment, depth int, noTree, noProjects bool) Plan {
	p := Plan{Version: "1", Home: e.Home, Warnings: append([]string{}, e.Inventory.Warnings...)}
	if !noTree {
		p.Tree = ScanTree(ctx, e, depth, 1000000)
	}
	seen := map[string]bool{}
	add := func(rule Rule, path, action, evidence string) {
		path = filepath.Clean(path)
		if seen[path] {
			return
		}
		seen[path] = true
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			return
		}
		candidate := Candidate{ID: rule.ID + ":" + path, Provider: rule.ID, Title: rule.Name + " cache", Path: path, Action: action, Scope: "user", Evidence: evidence, Status: "ready", Watch: rule.Watch}
		root, id, err := e.OpenCache(path)
		if err != nil {
			candidate.Status = "skipped"
			candidate.Reason = err.Error()
		} else {
			candidate.Identity = id
			candidate.Bytes, err = cacheSize(ctx, root, e.UID)
			root.Close()
			if err != nil {
				candidate.Status = "skipped"
				candidate.Reason = err.Error()
			}
		}
		if e.Active(rule.Watch) && len(rule.Watch) > 0 {
			candidate.Status = "skipped"
			candidate.Reason = "related application/build tool is running, or activity could not be checked"
		}
		if e.InUse(path) {
			candidate.Status = "skipped"
			candidate.Reason = "a running process is using this cache or a generated binary inside it"
		}
		p.Candidates = append(p.Candidates, candidate)
	}
	for _, rule := range Catalog {
		evidence := "documented cache artifact"
		for _, command := range rule.Commands {
			if e.Inventory.Commands[command] != "" {
				evidence = "installed executable: " + command
			}
		}
		for _, pkg := range rule.Packages {
			if e.Inventory.Packages[pkg] {
				evidence = "installed package: " + pkg
			}
		}
		for _, id := range rule.Flatpaks {
			if e.Inventory.Flatpaks[id] {
				evidence = "installed Flatpak: " + id
			}
		}
		for _, set := range []struct {
			root  string
			paths []string
		}{{e.Cache, rule.Cache}, {e.Config, rule.Config}, {e.Home, rule.Home}} {
			for _, pattern := range set.paths {
				paths, _ := filepath.Glob(filepath.Join(set.root, pattern))
				for _, path := range paths {
					add(rule, path, "directory", evidence)
				}
			}
		}
	}
	if e.Inventory.Commands["go"] != "" {
		rule := Rule{ID: "go", Name: "Go build", Watch: []string{"go", "compile", "link", "gopls"}}
		add(rule, filepath.Join(e.Cache, "go-build"), "go-build", "Go compiler cache")
		rule.ID = "go-modules"
		rule.Name = "Go module download"
		add(rule, filepath.Join(e.Home, "go/pkg/mod"), "go-modules", "default Go module cache")
	}
	if e.Inventory.Commands["uv"] != "" {
		add(Rule{ID: "uv", Name: "Unused Python uv", Watch: []string{"uv", "uvx", "pip", "pip3"}}, filepath.Join(e.Cache, "uv"), "uv", "uv prunes unused records; total size is an upper bound")
	}
	if !noProjects {
		home, err := os.OpenRoot(e.Home)
		if err == nil {
			_ = fs.WalkDir(home.FS(), ".", func(path string, d fs.DirEntry, err error) error {
				if ctx.Err() != nil {
					return fs.SkipAll
				}
				if err != nil {
					if d != nil && d.IsDir() {
						return fs.SkipDir
					}
					return nil
				}
				if !d.IsDir() || path == "." {
					return nil
				}
				base := d.Name()
				if strings.HasPrefix(base, ".") || base == "node_modules" || base == "vendor" {
					return fs.SkipDir
				}
				abs := filepath.Join(e.Home, filepath.FromSlash(path))
				for _, mount := range e.Mounts {
					if abs == mount {
						return fs.SkipDir
					}
				}
				if base == "target" {
					tag, readErr := home.ReadFile(filepath.Join(path, "CACHEDIR.TAG"))
					_, manifestErr := home.Stat(filepath.Join(filepath.Dir(path), "Cargo.toml"))
					if readErr == nil && manifestErr == nil && strings.HasPrefix(string(tag), "Signature: 8a477f597d28d172789f06886806bc55") {
						add(Rule{ID: "rust", Name: "Rust build", Watch: []string{"cargo", "rustc", "rust-analyzer"}}, abs, "directory", "Cargo cache-directory marker and adjacent manifest")
					}
					return fs.SkipDir
				}
				return nil
			})
			home.Close()
		}
	}
	for _, entry := range []struct{ id, command, title string }{{"docker-build", "docker", "Unused Docker build cache"}, {"apt-cache", "apt-get", "APT downloaded installers"}, {"apt-autoremove", "apt-get", "APT unused packages / obsolete kernels"}, {"snap-disabled", "snap", "Disabled Snap revisions"}, {"journals", "journalctl", "Archived system journals (14 days / 500 MB)"}} {
		if e.Inventory.Commands[entry.command] == "" {
			continue
		}
		scope := "system"
		if entry.id == "docker-build" {
			scope = "user"
		}
		p.Candidates = append(p.Candidates, Candidate{ID: entry.id, Provider: entry.command, Title: entry.title, Action: entry.id, Scope: scope, Evidence: "installed package manager / native cleanup API; size determined by tool", Status: "ready"})
		current := &p.Candidates[len(p.Candidates)-1]
		if entry.id == "snap-disabled" {
			var revisions []string
			for _, revision := range e.Inventory.Snaps {
				if revision.Disabled {
					revisions = append(revisions, revision.Name+"@"+revision.Revision)
				}
			}
			if len(revisions) == 0 {
				current.Status = "skipped"
				current.Reason = "no confirmed disabled Snap revisions"
			} else {
				current.Evidence = "disabled revisions: " + strings.Join(revisions, ", ")
			}
		}
		if entry.id == "apt-autoremove" {
			query, cancel := context.WithTimeout(ctx, 20*time.Second)
			output, err := e.Runner.Run(query, e.Inventory.Commands["apt-get"], "-s", "autoremove", "--purge")
			cancel()
			if err != nil {
				current.Status = "skipped"
				current.Reason = "APT removal plan could not be verified"
			} else {
				var removals, kernels []string
				for _, line := range strings.Split(output, "\n") {
					fields := strings.Fields(line)
					if len(fields) > 1 && (fields[0] == "Remv" || fields[0] == "Purg") {
						removals = append(removals, fields[1])
					}
				}
				for name := range e.Inventory.Packages {
					if strings.HasPrefix(name, "linux-image-") {
						kernels = append(kernels, "ii "+name)
					}
				}
				if len(removals) == 0 {
					current.Status = "skipped"
					current.Reason = "no package-manager-confirmed unused packages"
				} else if guardErr := KernelGuard(output, strings.Join(kernels, "\n"), RunningKernel()); guardErr != nil {
					current.Status = "skipped"
					current.Reason = guardErr.Error()
				} else {
					current.Evidence = "APT proposed removals: " + strings.Join(removals, ", ")
				}
			}
		}
		if entry.id == "docker-build" {
			if err := LocalDocker(ctx, e); err != nil {
				p.Candidates[len(p.Candidates)-1].Status = "skipped"
				p.Candidates[len(p.Candidates)-1].Reason = err.Error()
			}
		}
	}
	sort.Slice(p.Candidates, func(i, j int) bool { return p.Candidates[i].ID < p.Candidates[j].ID })
	return p
}
