package core

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

func (e Environment) OpenCache(path string) (*os.Root, Identity, error) {
	resolvedHome, err := filepath.EvalSymlinks(e.Home)
	if err != nil {
		return nil, Identity{}, err
	}
	if resolvedHome != filepath.Clean(e.Home) {
		return nil, Identity{}, fmt.Errorf("unresolved home alias cannot authorize cache cleanup")
	}
	path = filepath.Clean(path)
	if path == e.Home || !within(e.Home, path) {
		return nil, Identity{}, fmt.Errorf("cache is outside the supported home boundary")
	}
	if e.MountedBelow(path) {
		return nil, Identity{}, fmt.Errorf("cache intersects a mounted filesystem")
	}
	home, err := os.OpenRoot(e.Home)
	if err != nil {
		return nil, Identity{}, err
	}
	defer home.Close()
	rel, _ := filepath.Rel(e.Home, path)
	part := ""
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		part = filepath.Join(part, component)
		info, err := home.Lstat(part)
		if err != nil {
			return nil, Identity{}, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, Identity{}, fmt.Errorf("symlinked cache path is inspection-only")
		}
	}
	info, err := home.Lstat(rel)
	if err != nil {
		return nil, Identity{}, err
	}
	want := identity(info)
	if !info.IsDir() || want.UID != e.UID {
		return nil, Identity{}, fmt.Errorf("cache is not a directory owned by this account")
	}
	root, err := home.OpenRoot(rel)
	if err != nil {
		return nil, Identity{}, err
	}
	actual, err := root.Stat(".")
	if err != nil || identity(actual) != want {
		root.Close()
		return nil, Identity{}, fmt.Errorf("cache identity changed during inspection")
	}
	return root, want, nil
}

func cacheSize(ctx context.Context, root *os.Root, uid uint32) (int64, error) {
	var bytes int64
	seen := map[[2]uint64]bool{}
	err := fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if identity(info).UID != uid {
			return fmt.Errorf("contains entries owned by another account")
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("contains an active socket or special file")
		}
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			key := [2]uint64{uint64(stat.Dev), stat.Ino}
			if !info.IsDir() && stat.Nlink <= 1 && !seen[key] {
				bytes += stat.Blocks * 512
				seen[key] = true
			}
		}
		return nil
	})
	return bytes, err
}

func cacheWritable(root *os.Root, uid uint32) error {
	return fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if identity(info).UID != uid {
			return fmt.Errorf("cache directory ownership changed")
		}
		if info.Mode().Perm()&0200 != 0 {
			return nil
		}
		file, err := root.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		opened, err := file.Stat()
		if err != nil {
			return err
		}
		if identity(opened) != identity(info) || !opened.IsDir() {
			return fmt.Errorf("cache directory changed before permission adjustment")
		}
		return file.Chmod(opened.Mode().Perm() | 0700)
	})
}

func ScanTree(ctx context.Context, e Environment, depth, limit int) Tree {
	tree := Tree{Root: e.Home}
	root, err := os.OpenRoot(e.Home)
	if err != nil {
		tree.Warnings++
		return tree
	}
	defer root.Close()
	nodes := map[string]int64{".": 0}
	seen := map[[2]uint64]bool{}
	entries := 0
	_ = fs.WalkDir(root.FS(), ".", func(path string, d fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return fs.SkipAll
		}
		entries++
		if entries > limit {
			tree.Limited = true
			return fs.SkipAll
		}
		if walkErr != nil {
			tree.Warnings++
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if path != "." && d.IsDir() {
			absolute := filepath.Join(e.Home, filepath.FromSlash(path))
			for _, point := range e.Mounts {
				if absolute == point {
					tree.Warnings++
					return fs.SkipDir
				}
			}
			if strings.Count(path, "/") < depth {
				nodes[path] += 0
			}
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			tree.Warnings++
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		tree.Files++
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return nil
		}
		key := [2]uint64{uint64(stat.Dev), stat.Ino}
		if seen[key] {
			return nil
		}
		seen[key] = true
		for parent := filepath.ToSlash(filepath.Dir(path)); ; parent = filepath.ToSlash(filepath.Dir(parent)) {
			if parent == "." || strings.Count(parent, "/") < depth {
				nodes[parent] += stat.Blocks * 512
			}
			if parent == "." {
				break
			}
		}
		return nil
	})
	for path, bytes := range nodes {
		tree.Nodes = append(tree.Nodes, TreeNode{Path: path, Bytes: bytes})
	}
	sort.Slice(tree.Nodes, func(i, j int) bool { return tree.Nodes[i].Path < tree.Nodes[j].Path })
	return tree
}
