package app

import (
	"bytes"
	"context"
	"github.com/arnavsx3/Eversor/internal/core"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIHelpVersionAndInvalidArguments(t *testing.T) {
	for _, args := range [][]string{{}, {"--help"}, {"version"}} {
		var out, errors bytes.Buffer
		if Run(context.Background(), args, strings.NewReader(""), &out, &errors) != 0 || out.Len() == 0 {
			t.Fatalf("command failed: %v %s", args, errors.String())
		}
	}
	for _, args := range [][]string{{"delete"}, {"dry-run", "--tree-depth", "0"}, {"dry-run", "unexpected"}} {
		var out, errors bytes.Buffer
		if Run(context.Background(), args, strings.NewReader(""), &out, &errors) != 2 {
			t.Fatalf("invalid arguments accepted: %v", args)
		}
	}
}

func TestDryRunCancellationAndCleanAgainstDisposableHome(t *testing.T) {
	home := t.TempDir()
	cache := filepath.Join(home, ".cache/pip")
	if err := os.MkdirAll(cache, 0700); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(cache, "blob")
	if err := os.WriteFile(file, []byte("cache"), 0600); err != nil {
		t.Fatal(err)
	}
	important := filepath.Join(home, "important")
	if err := os.WriteFile(important, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_RUNTIME_DIR", t.TempDir())
	previous := discover
	defer func() { discover = previous }()
	discover = func(context.Context) (core.Environment, error) {
		return core.Environment{Home: home, Cache: filepath.Join(home, ".cache"), Config: filepath.Join(home, ".config"), Data: filepath.Join(home, ".local/share"), UID: uint32(os.Getuid()), ProcessesKnown: true, Inventory: core.Inventory{Commands: map[string]string{}, Packages: map[string]bool{}, Flatpaks: map[string]bool{}}}, nil
	}
	for _, args := range [][]string{{"dry-run", "--json", "--no-tree"}, {"clean", "--user-only", "--no-tree"}} {
		var output, errors bytes.Buffer
		if status := Run(context.Background(), args, strings.NewReader("no\n"), &output, &errors); status != 0 {
			t.Fatalf("preview/cancel failed %d: %s", status, errors.String())
		}
		if _, err := os.Stat(file); err != nil {
			t.Fatal("preview or cancellation deleted cache")
		}
	}
	var output, errors bytes.Buffer
	if status := Run(context.Background(), []string{"clean", "--user-only", "--no-tree", "--yes", "--json"}, strings.NewReader(""), &output, &errors); status != 0 {
		t.Fatalf("fixture clean failed %d: %s %s", status, errors.String(), output.String())
	}
	if _, err := os.Stat(file); !os.IsNotExist(err) {
		t.Fatal("eligible fixture cache not removed")
	}
	if data, err := os.ReadFile(important); err != nil || string(data) != "keep" {
		t.Fatal("personal fixture changed")
	}
}
