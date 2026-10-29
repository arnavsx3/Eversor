package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

type Refresh func(Environment) (Environment, error)

func RefreshLive(e Environment) (Environment, error) {
	var err error
	e.Mounts, err = MountPoints()
	if err != nil {
		return e, err
	}
	e.Processes, e.ProcessesKnown = Processes(e.UID)
	return e, nil
}

func approved(e Environment, c Candidate) ([]string, bool) {
	if c.ID == "docker-build" && c.Action == "docker-build" && c.Path == "" {
		return []string{"docker-build"}, true
	}
	if c.Action == "go-build" && c.Provider == "go" && c.Path == filepath.Join(e.Cache, "go-build") {
		return []string{"go", "compile", "link", "gopls"}, true
	}
	if c.Action == "go-modules" && c.Provider == "go-modules" && c.Path == filepath.Join(e.Home, "go/pkg/mod") {
		return []string{"go", "compile", "link", "gopls"}, true
	}
	if c.Action == "uv" && c.Provider == "uv" && c.Path == filepath.Join(e.Cache, "uv") {
		return []string{"uv", "uvx", "pip", "pip3"}, true
	}
	if c.Action != "directory" {
		return nil, false
	}
	for _, rule := range Catalog {
		if rule.ID != c.Provider {
			continue
		}
		for _, set := range []struct {
			root  string
			paths []string
		}{{e.Cache, rule.Cache}, {e.Config, rule.Config}, {e.Home, rule.Home}} {
			for _, pattern := range set.paths {
				if matches, _ := filepath.Match(filepath.Join(set.root, pattern), c.Path); matches {
					return rule.Watch, true
				}
			}
		}
	}
	if c.Provider == "rust" && filepath.Base(c.Path) == "target" && within(e.Home, c.Path) {
		home, err := os.OpenRoot(e.Home)
		if err != nil {
			return nil, false
		}
		defer home.Close()
		rel, _ := filepath.Rel(e.Home, c.Path)
		tag, err := home.ReadFile(filepath.Join(rel, "CACHEDIR.TAG"))
		if err != nil {
			return nil, false
		}
		info, err := home.Lstat(filepath.Join(filepath.Dir(rel), "Cargo.toml"))
		if err == nil && info.Mode().IsRegular() && strings.HasPrefix(string(tag), "Signature: 8a477f597d28d172789f06886806bc55") {
			return []string{"cargo", "rustc", "rust-analyzer"}, true
		}
	}
	return nil, false
}

func CleanCandidate(ctx context.Context, e Environment, c Candidate, refresh Refresh) Result {
	result := Result{ID: c.ID, Title: c.Title, Status: "skipped"}
	if c.Status != "ready" {
		result.Detail = c.Reason
		return result
	}
	watch, ok := approved(e, c)
	if !ok {
		result.Detail = "no supported cleanup rule authorizes this action"
		return result
	}
	if c.Scope != "user" {
		result.Detail = "system actions require the fixed system helper"
		return result
	}
	var err error
	e, err = refresh(e)
	if err != nil {
		result.Detail = "could not recheck activity/mounts: " + err.Error()
		return result
	}
	if !e.ProcessesKnown || (len(watch) > 0 && e.Active(watch)) {
		result.Detail = "related application/build tool is active, or activity is unknown"
		return result
	}
	if e.InUse(c.Path) {
		result.Detail = "a running process is using this cache"
		return result
	}
	if ctx.Err() != nil {
		result.Status = "failed"
		result.Detail = ctx.Err().Error()
		return result
	}
	if c.Action == "docker-build" {
		var host string
		host, err = DockerEndpoint(ctx, e)
		if err != nil {
			result.Detail = err.Error()
			return result
		}
		result.Detail, err = e.Runner.Run(ctx, "/usr/bin/env", "-u", "DOCKER_CONTEXT", "-u", "DOCKER_HOST", e.Inventory.Commands["docker"], "--host", host, "builder", "prune", "--all", "--force")
	} else {
		root, id, openErr := e.OpenCache(c.Path)
		if openErr != nil {
			result.Detail = "cache no longer eligible: " + openErr.Error()
			return result
		}
		defer root.Close()
		if id != c.Identity {
			result.Detail = "cache was replaced after inspection"
			return result
		}
		if _, err = cacheSize(ctx, root, e.UID); err != nil {
			result.Detail = "cache contents no longer eligible: " + err.Error()
			return result
		}
		switch c.Action {
		case "directory", "go-build", "go-modules":
			// Go module-cache directories can be read-only. Change permissions
			// only through verified open directory handles inside this cache.
			if err = cacheWritable(root, e.UID); err != nil {
				break
			}
			dir, openErr := root.Open(".")
			err = openErr
			if err == nil {
				var entries []os.DirEntry
				entries, err = dir.ReadDir(-1)
				dir.Close()
				if err == nil {
					for _, entry := range entries {
						if ctx.Err() != nil {
							err = ctx.Err()
							break
						}
						if err = root.RemoveAll(entry.Name()); err != nil {
							break
						}
					}
				}
			}
			if err == nil {
				result.Detail = "cleared approved cache contents; retained the cache directory"
			}
		case "uv":
			pinned, ok := e.Runner.(PinnedRunner)
			if !ok {
				err = errors.New("runner cannot pin the cache directory")
				break
			}
			directory, openErr := root.Open(".")
			if openErr != nil {
				err = openErr
				break
			}
			result.Detail, err = pinned.RunPinned(ctx, directory, e.Inventory.Commands["uv"], "cache", "prune", "--cache-dir", "/proc/self/fd/3")
			directory.Close()
		default:
			err = errors.New("unsupported user action")
		}
	}
	if err != nil {
		result.Status = "failed"
		result.Detail = strings.TrimSpace(result.Detail + "\n" + err.Error())
	} else {
		result.Status = "cleaned"
	}
	return result
}

func LocalDocker(ctx context.Context, e Environment) error {
	_, err := DockerEndpoint(ctx, e)
	return err
}

func DockerEndpoint(ctx context.Context, e Environment) (string, error) {
	if e.Inventory.Commands["docker"] == "" {
		return "", errors.New("Docker executable unavailable")
	}
	if host := os.Getenv("DOCKER_HOST"); host != "" && !strings.HasPrefix(host, "unix://") {
		return "", errors.New("remote Docker endpoint is inspection-only")
	}
	output, err := e.Runner.Run(ctx, e.Inventory.Commands["docker"], "context", "inspect", "--format", "{{json .Endpoints.docker.Host}}")
	if err != nil {
		return "", errors.New("Docker context could not be verified; skipped")
	}
	var host string
	if json.Unmarshal([]byte(strings.TrimSpace(output)), &host) != nil || !strings.HasPrefix(host, "unix://") {
		return "", errors.New("only a verified local Unix-socket Docker endpoint may be cleaned")
	}
	return host, nil
}

func AcquireLock(e Environment) (func(), error) {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if !filepath.IsAbs(base) {
		base = os.TempDir()
	}
	path := filepath.Join(base, fmt.Sprintf("eversor-%d.lock", e.UID))
	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	info, err := file.Stat()
	if err != nil || identity(info).UID != e.UID || !info.Mode().IsRegular() {
		file.Close()
		return nil, errors.New("unsafe lock-file ownership or type")
	}
	if err = syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, errors.New("another Eversor cleanup is running")
	}
	return func() { file.Close() }, nil
}

func AvailableBytes() int64 { return diskAvailable() }
