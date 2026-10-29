package core

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func commandEnvironment() []string {
	var result []string
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "LC_ALL=") {
			result = append(result, entry)
		}
	}
	return append(result, "LC_ALL=C")
}

func Discover(ctx context.Context) (Environment, error) {
	account, err := user.Current()
	if err != nil {
		return Environment{}, err
	}
	uid, err := strconv.ParseUint(account.Uid, 10, 32)
	if err != nil {
		return Environment{}, err
	}
	if uid == 0 {
		return Environment{}, errors.New("run Eversor from your normal account; system actions elevate separately")
	}
	home := filepath.Clean(account.HomeDir)
	xdg := func(name, fallback string) string {
		if value := os.Getenv(name); filepath.IsAbs(value) {
			return filepath.Clean(value)
		}
		return filepath.Join(home, fallback)
	}
	e := Environment{Home: home, Cache: xdg("XDG_CACHE_HOME", ".cache"),
		Config: xdg("XDG_CONFIG_HOME", ".config"), Data: xdg("XDG_DATA_HOME", ".local/share"),
		UID: uint32(uid), Runner: ExecRunner{}}
	e.Mounts, err = MountPoints()
	if err != nil {
		return e, fmt.Errorf("read mount boundaries: %w", err)
	}
	e.Processes, e.ProcessesKnown = Processes(e.UID)
	e.Inventory = Inventory{Commands: map[string]string{}, Packages: map[string]bool{}, Flatpaks: map[string]bool{}}
	for _, name := range []string{"docker", "go", "npm", "uv", "trivy", "cargo", "kubectl", "helm", "apt-get", "snap", "flatpak", "journalctl", "code", "codium", "google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "brave-browser", "firefox", "slack", "discord", "spotify", "claude", "codex", "chatgpt", "antigravity", "pip", "pip3", "yarn", "pnpm", "gradle", "mvn"} {
		if path, lookupErr := exec.LookPath(name); lookupErr == nil {
			e.Inventory.Commands[name] = path
		}
	}
	query := func(name string, args ...string) string {
		path := e.Inventory.Commands[name]
		if name == "dpkg-query" {
			path = "/usr/bin/dpkg-query"
		}
		if path == "" {
			return ""
		}
		qctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
		out, queryErr := e.Runner.Run(qctx, path, args...)
		if queryErr != nil {
			e.Inventory.Warnings = append(e.Inventory.Warnings, name+" inventory unavailable")
			return ""
		}
		return out
	}
	for _, line := range strings.Split(query("dpkg-query", "-W", "-f=${binary:Package}\t${db:Status-Abbrev}\n"), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) == 2 && strings.HasPrefix(parts[1], "ii") {
			e.Inventory.Packages[strings.Split(parts[0], ":")[0]] = true
		}
	}
	e.Inventory.Snaps = ParseSnaps(query("snap", "list", "--all"))
	for _, id := range strings.Fields(query("flatpak", "list", "--app", "--columns=application")) {
		e.Inventory.Flatpaks[id] = true
	}
	return e, nil
}

func Processes(uid uint32) ([]Process, bool) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, false
	}
	known := true
	var result []Process
	for _, entry := range entries {
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		path := filepath.Join("/proc", entry.Name())
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != uid {
			continue
		}
		name, nerr := os.ReadFile(filepath.Join(path, "comm"))
		args, aerr := os.ReadFile(filepath.Join(path, "cmdline"))
		if os.IsPermission(nerr) || os.IsPermission(aerr) {
			known = false
		}
		if nerr != nil {
			continue
		}
		process := Process{Name: strings.ToLower(strings.TrimSpace(string(name))), Args: strings.Split(string(args), "\x00")}
		links := []string{filepath.Join(path, "exe"), filepath.Join(path, "cwd")}
		if files, readErr := os.ReadDir(filepath.Join(path, "fd")); readErr == nil {
			for _, file := range files {
				links = append(links, filepath.Join(path, "fd", file.Name()))
			}
		}
		for _, link := range links {
			if target, linkErr := os.Readlink(link); linkErr == nil && filepath.IsAbs(target) {
				process.Paths = append(process.Paths, strings.TrimSuffix(target, " (deleted)"))
			}
		}
		result = append(result, process)
	}
	return result, known
}

func (e Environment) Active(names []string) bool {
	if !e.ProcessesKnown {
		return true
	}
	for _, process := range e.Processes {
		for _, name := range names {
			if process.Name == strings.ToLower(name) {
				return true
			}
			for _, arg := range process.Args {
				base := filepath.Base(arg)
				if base == name || (name == "npm" && base == "npm-cli.js") || (name == "pre-commit" && arg == "pre_commit") {
					return true
				}
			}
		}
	}
	return false
}

func (e Environment) InUse(path string) bool {
	if path == "" {
		return false
	}
	for _, process := range e.Processes {
		for _, opened := range process.Paths {
			if within(path, opened) {
				return true
			}
		}
	}
	return false
}

func MountPoints() ([]string, error) {
	file, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var points []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 {
			continue
		}
		replacer := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
		points = append(points, filepath.Clean(replacer.Replace(fields[4])))
	}
	return points, scanner.Err()
}

func within(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (e Environment) MountedBelow(path string) bool {
	for _, point := range e.Mounts {
		if within(path, point) {
			return true
		}
		if within(e.Home, point) && within(point, path) {
			return true
		}
	}
	return false
}

func identity(info os.FileInfo) Identity {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return Identity{Device: uint64(stat.Dev), Inode: stat.Ino, UID: stat.Uid}
	}
	return Identity{}
}

func diskAvailable() int64 {
	var stat syscall.Statfs_t
	if syscall.Statfs("/", &stat) != nil {
		return 0
	}
	return int64(stat.Bavail) * stat.Bsize
}
