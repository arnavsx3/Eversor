package app

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/arnavsx3/Eversor/internal/core"
	"github.com/arnavsx3/Eversor/internal/report"
)

var Version = "dev"
var Revision = "unknown"

var discover = core.Discover

func Run(ctx context.Context, args []string, in io.Reader, out, errout io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprintln(out, "Eversor\n\nUsage: eversor dry-run [options]\n       eversor clean [options]\n       eversor version\n\nOptions:\n  --json          emit machine-readable output\n  --tree          include the full home storage tree (slower)\n  --no-tree       omit the home storage tree (default)\n  --tree-depth N  tree depth (1-4; default 2)\n  --no-projects   retain project build outputs\n  --user-only     omit privileged package/log cleanup\n  --yes           confirm clean without an interactive question\n\nOnly supported caches and package-manager-confirmed obsolete items are eligible.\nUnknown folders, profiles, credentials, source code, databases and installed toolchains are retained.")
		return 0
	}
	if args[0] == "version" || args[0] == "--version" {
		if Revision != "unknown" {
			fmt.Fprintf(out, "eversor %s (%s)\n", Version, Revision)
		} else {
			fmt.Fprintln(out, "eversor", Version)
		}
		return 0
	}
	if args[0] == "__system-clean" {
		return systemEndpoint(ctx, args[1:], out, errout)
	}
	if args[0] != "dry-run" && args[0] != "clean" {
		fmt.Fprintln(errout, "unknown command; use eversor --help")
		return 2
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(errout)
	jsonOutput := flags.Bool("json", false, "emit JSON")
	noTree := flags.Bool("no-tree", false, "omit the home storage tree (default)")
	showTree := flags.Bool("tree", false, "include the full home storage tree")
	noProjects := flags.Bool("no-projects", false, "retain project build outputs")
	userOnly := flags.Bool("user-only", false, "omit privileged cleanup")
	yes := flags.Bool("yes", false, "confirm cleanup")
	depth := flags.Int("tree-depth", 2, "storage tree depth (1-4)")
	if err := flags.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *depth < 1 || *depth > 4 {
		fmt.Fprintln(errout, "unexpected arguments or invalid tree depth")
		return 2
	}
	wantTree := *showTree
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "tree-depth" {
			wantTree = true
		}
	})
	if wantTree && *noTree {
		fmt.Fprintln(errout, "--no-tree cannot be combined with --tree or --tree-depth")
		return 2
	}
	if args[0] == "clean" && *jsonOutput && !*yes {
		fmt.Fprintln(errout, "clean --json requires --yes; preview first with dry-run --json")
		return 2
	}
	if !*jsonOutput {
		fmt.Fprintln(errout, "Inspecting supported caches…")
	}
	env, err := discover(ctx)
	if err != nil {
		fmt.Fprintln(errout, err)
		return 1
	}
	plan := core.BuildPlan(ctx, env, *depth, !wantTree, *noProjects)
	if *userOnly {
		filtered := plan.Candidates[:0]
		for _, c := range plan.Candidates {
			if c.Scope == "user" {
				filtered = append(filtered, c)
			}
		}
		plan.Candidates = filtered
	}
	if ctx.Err() != nil {
		fmt.Fprintln(errout, ctx.Err())
		return 1
	}
	if args[0] == "dry-run" {
		if *jsonOutput {
			return encode(out, plan)
		}
		report.Plan(out, plan)
		return 0
	}
	if !*jsonOutput {
		report.Plan(out, plan)
	}
	if !*yes {
		fmt.Fprint(out, "\nClean the eligible items above? [y/N] ")
		confirmed, readErr := confirm(ctx, in)
		if readErr != nil {
			fmt.Fprintln(errout, readErr)
			return 1
		}
		if !confirmed {
			fmt.Fprintln(out, "Cancelled. No cleanup actions executed.")
			return 0
		}
	}
	unlock, err := core.AcquireLock(env)
	if err != nil {
		fmt.Fprintln(errout, err)
		return 1
	}
	defer unlock()
	before := core.MeasureSpace(env, plan)
	r := core.Report{Plan: plan}
	var system []string
	for _, c := range plan.Candidates {
		if ctx.Err() != nil {
			r.Results = append(r.Results, core.Result{ID: c.ID, Title: c.Title, Status: "failed", Detail: "interrupted"})
			break
		}
		if c.Scope == "system" && c.Status == "ready" {
			system = append(system, c.Action)
			continue
		}
		if !*jsonOutput {
			fmt.Fprintln(out, "\nCleaning:", c.Title)
		}
		result := core.CleanCandidate(ctx, env, c, core.RefreshLive)
		r.Results = append(r.Results, result)
		if !*jsonOutput {
			report.Result(out, result)
		}
	}
	if len(system) > 0 && ctx.Err() == nil {
		if !*jsonOutput {
			fmt.Fprintln(out, "\nSystem package/log cleanup — sudo may request your password.")
		}
		results, err := elevate(ctx, system, in, errout)
		if err != nil {
			for _, id := range system {
				results = append(results, core.Result{ID: id, Title: id, Status: "failed", Detail: err.Error()})
			}
		}
		r.Results = append(r.Results, results...)
		if !*jsonOutput {
			for _, result := range results {
				report.Result(out, result)
			}
		}
	}
	r.Filesystems, r.MeasurementWarnings = before.Change()
	for _, filesystem := range r.Filesystems {
		r.FreedBytes += filesystem.Bytes
	}
	status := 0
	r.Interrupted = ctx.Err() != nil
	if r.Interrupted {
		status = 1
	}
	for _, result := range r.Results {
		if result.Status == "failed" {
			status = 1
		}
	}
	if *jsonOutput {
		if encode(out, r) != 0 {
			return 1
		}
	} else {
		report.Summary(out, r)
	}
	return status
}

func confirm(ctx context.Context, in io.Reader) (bool, error) {
	type answer struct {
		line string
		err  error
	}
	answers := make(chan answer, 1)
	go func() {
		line, err := bufio.NewReader(in).ReadString('\n')
		answers <- answer{line, err}
	}()
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case response := <-answers:
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if response.err == io.EOF {
			return false, nil
		}
		if response.err != nil {
			return false, response.err
		}
		line := strings.ToLower(strings.TrimSpace(response.line))
		return line == "y" || line == "yes", nil
	}
}

func encode(out io.Writer, value any) int {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if encoder.Encode(value) != nil {
		return 1
	}
	return 0
}

func systemEndpoint(ctx context.Context, ids []string, out, errout io.Writer) int {
	if os.Geteuid() != 0 {
		fmt.Fprintln(errout, "system helper requires administrator authorization")
		return 1
	}
	if len(ids) == 0 {
		fmt.Fprintln(errout, "no system actions requested")
		return 2
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !core.SystemActions[id] || seen[id] {
			fmt.Fprintln(errout, "unsupported or repeated system action")
			return 2
		}
		seen[id] = true
	}
	return encode(out, core.RunSystem(ctx, ids, core.ExecRunner{System: true}, core.RunningKernel()))
}

func elevate(ctx context.Context, ids []string, in io.Reader, errout io.Writer) ([]core.Result, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return nil, err
	}
	// Refuse to elevate a development binary that the ordinary user can replace.
	for path := executable; ; path = filepath.Dir(path) {
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != 0 || info.Mode().Perm()&0022 != 0 {
			return nil, fmt.Errorf("install a root-owned binary in /usr/local/bin for system cleanup, or use --user-only")
		}
		if path == "/" {
			break
		}
	}
	args := append([]string{"--", executable, "__system-clean"}, ids...)
	command := exec.CommandContext(ctx, "/usr/bin/sudo", args...)
	command.Stdin = in
	command.Stderr = errout
	data, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("system cleanup was not completed: %w", err)
	}
	var result []core.Result
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid system helper response: %w", err)
	}
	if len(result) != len(ids) {
		return nil, fmt.Errorf("incomplete system helper response")
	}
	return result, nil
}
