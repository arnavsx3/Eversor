package app

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"

	"github.com/arnavsx3/Eversor/internal/core"
	"github.com/arnavsx3/Eversor/internal/report"
)

var Version = "dev"

func Run(ctx context.Context, args []string, in io.Reader, out, errout io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprintln(out, "Eversor\n\nUsage: eversor dry-run [--json] [--no-tree] [--no-projects]\n       eversor version\n\nInspect known app caches. Unknown folders are never cleanup targets.")
		return 0
	}
	if args[0] == "version" || args[0] == "--version" {
		fmt.Fprintln(out, "eversor", Version)
		return 0
	}
	if args[0] != "dry-run" {
		fmt.Fprintln(errout, "unknown command; use eversor --help")
		return 2
	}
	flags := flag.NewFlagSet("dry-run", flag.ContinueOnError)
	flags.SetOutput(errout)
	jsonOutput := flags.Bool("json", false, "emit a JSON plan")
	noTree := flags.Bool("no-tree", false, "omit the home storage tree")
	noProjects := flags.Bool("no-projects", false, "omit project build outputs")
	depth := flags.Int("tree-depth", 2, "storage tree depth (1-4)")
	if flags.Parse(args[1:]) != nil {
		return 2
	}
	if flags.NArg() != 0 || *depth < 1 || *depth > 4 {
		fmt.Fprintln(errout, "unexpected arguments or invalid tree depth")
		return 2
	}
	env, err := core.Discover(ctx)
	if err != nil {
		fmt.Fprintln(errout, err)
		return 1
	}
	plan := core.BuildPlan(ctx, env, *depth, *noTree, *noProjects)
	if ctx.Err() != nil {
		fmt.Fprintln(errout, ctx.Err())
		return 1
	}
	if *jsonOutput {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if encoder.Encode(plan) != nil {
			return 1
		}
	} else {
		report.Plan(out, plan)
	}
	return 0
}
