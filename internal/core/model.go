// Package core contains Eversor's inventory, provider rules and cleanup policy.
package core

import (
	"context"
	"os"
	"os/exec"
	"time"
)

type Runner interface {
	Run(context.Context, string, ...string) (string, error)
}

type ExecRunner struct{ System bool }

type PinnedRunner interface {
	RunPinned(context.Context, *os.File, string, ...string) (string, error)
}

func (r ExecRunner) RunPinned(ctx context.Context, directory *os.File, name string, args ...string) (string, error) {
	query, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(query, name, args...)
	cmd.Env = commandEnvironment()
	cmd.ExtraFiles = []*os.File{directory}
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (r ExecRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	query, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(query, name, args...)
	cmd.Env = commandEnvironment()
	if r.System {
		cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LC_ALL=C", "DEBIAN_FRONTEND=noninteractive"}
	}
	output, err := cmd.CombinedOutput()
	return string(output), err
}

type Process struct {
	Name  string
	Args  []string
	Paths []string
}

type Inventory struct {
	Commands map[string]string `json:"commands"`
	Packages map[string]bool   `json:"-"`
	Flatpaks map[string]bool   `json:"-"`
	Snaps    []SnapRevision    `json:"-"`
	Warnings []string          `json:"warnings,omitempty"`
}

type Environment struct {
	Home, Cache, Config, Data string
	UID                       uint32
	Mounts                    []string
	Processes                 []Process
	ProcessesKnown            bool
	Inventory                 Inventory
	Runner                    Runner
}

type Identity struct {
	Device uint64
	Inode  uint64
	UID    uint32
}

type Candidate struct {
	ID       string   `json:"id"`
	Provider string   `json:"provider"`
	Title    string   `json:"title"`
	Path     string   `json:"path,omitempty"`
	Action   string   `json:"action"`
	Scope    string   `json:"scope"`
	Evidence string   `json:"evidence"`
	Bytes    int64    `json:"estimated_bytes"`
	Status   string   `json:"status"`
	Reason   string   `json:"reason,omitempty"`
	Identity Identity `json:"-"`
	Watch    []string `json:"-"`
}

type TreeNode struct {
	Path  string `json:"path"`
	Bytes int64  `json:"allocated_bytes"`
}

type Tree struct {
	Root     string     `json:"root"`
	Nodes    []TreeNode `json:"nodes"`
	Files    int        `json:"files"`
	Warnings int        `json:"unreadable_or_excluded_entries"`
	Limited  bool       `json:"entry_limit_reached"`
}

type Plan struct {
	Version    string      `json:"version"`
	Home       string      `json:"home"`
	Tree       Tree        `json:"tree"`
	Candidates []Candidate `json:"candidates"`
	Warnings   []string    `json:"warnings,omitempty"`
}

type Result struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type Report struct {
	Plan                Plan               `json:"plan"`
	Results             []Result           `json:"results"`
	FreedBytes          int64              `json:"filesystem_available_change_bytes"`
	Filesystems         []FilesystemChange `json:"filesystems,omitempty"`
	MeasurementWarnings []string           `json:"measurement_warnings,omitempty"`
}
