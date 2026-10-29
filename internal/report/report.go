package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/arnavsx3/Eversor/internal/core"
)

func Size(value int64) string {
	n := float64(value)
	for _, unit := range []string{"B", "KB", "MB", "GB", "TB"} {
		if n < 1000 && n > -1000 || unit == "TB" {
			return fmt.Sprintf("%.1f %s", n, unit)
		}
		n /= 1000
	}
	return ""
}

func Plan(w io.Writer, p core.Plan) {
	fmt.Fprintln(w, "Eversor — cleanup preview")
	if p.Tree.Root != "" {
		fmt.Fprintf(w, "\nStorage tree: %s (unknown folders are inspection-only)\n", p.Tree.Root)
		for _, node := range p.Tree.Nodes {
			fmt.Fprintf(w, "  %s%-48s %10s\n", strings.Repeat("  ", strings.Count(node.Path, "/")), node.Path, Size(node.Bytes))
		}
		fmt.Fprintf(w, "  %d files inspected; %d excluded/unreadable entries\n", p.Tree.Files, p.Tree.Warnings)
		if p.Tree.Limited {
			fmt.Fprintln(w, "  Entry limit reached; tree is partial.")
		}
	}
	fmt.Fprintln(w, "\nSupported cleanup candidates:")
	var bytes int64
	for _, c := range p.Candidates {
		size := "tool-managed"
		if c.Path != "" {
			size = Size(c.Bytes)
		}
		fmt.Fprintf(w, "  [%s] %s — %s (%s)\n", c.Status, c.Title, size, c.Scope)
		if c.Path != "" {
			fmt.Fprintf(w, "    %s\n", c.Path)
		}
		if c.Reason != "" {
			fmt.Fprintf(w, "    Skipped: %s\n", c.Reason)
		} else {
			bytes += c.Bytes
			fmt.Fprintf(w, "    %s\n", c.Evidence)
		}
	}
	fmt.Fprintf(w, "\nEligible inspected cache data: %s. Native-tool recovery is determined at cleanup.\n", Size(bytes))
	fmt.Fprintln(w, "Estimates exclude shared hard-linked files. No files were deleted by this inspection.")
	for _, warning := range p.Warnings {
		fmt.Fprintln(w, "Inventory note:", warning)
	}
}
