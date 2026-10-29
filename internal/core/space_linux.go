package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"
)

type filesystemID struct {
	Type   int64
	ID     syscall.Fsid
	Device uint64 // Fallback for filesystems without an fsid.
}

type diskSample struct {
	ID        filesystemID
	Path      string
	Available int64
}

type FilesystemChange struct {
	Path  string `json:"representative_path"`
	Bytes int64  `json:"available_change_bytes"`
}

type SpaceSnapshot struct {
	samples  map[filesystemID]diskSample
	warnings []string
}

func inspectDisk(path string) (diskSample, error) {
	var fs syscall.Statfs_t
	if err := syscall.Statfs(path, &fs); err != nil {
		return diskSample{}, err
	}
	key := filesystemID{Type: int64(fs.Type), ID: fs.Fsid}
	if fs.Fsid == (syscall.Fsid{}) {
		info, err := os.Stat(path)
		if err != nil {
			return diskSample{}, err
		}
		key.Device = identity(info).Device
	}
	return diskSample{ID: key, Path: path, Available: int64(fs.Bavail) * fs.Bsize}, nil
}

// MeasureSpace samples relevant filesystems once, even when many cache paths or
// bind mounts share their available space. It never scans directory contents.
func MeasureSpace(e Environment, p Plan) SpaceSnapshot {
	paths := []string{"/", e.Home}
	for _, c := range p.Candidates {
		if c.Status != "ready" {
			continue
		}
		if c.Path != "" {
			paths = append(paths, c.Path)
			continue
		}
		switch c.Action {
		case "apt-cache":
			paths = append(paths, "/var/cache/apt/archives")
		case "apt-autoremove":
			paths = append(paths, "/usr", "/boot", "/var")
		case "journals":
			paths = append(paths, "/var/log/journal", "/run/log/journal")
		case "snap-disabled":
			paths = append(paths, "/var/lib/snapd/snaps")
		case "docker-build":
			paths = append(paths, "/var/lib/docker", filepath.Join(e.Data, "docker"))
		}
	}
	return sampleSpace(paths, inspectDisk)
}

func sampleSpace(paths []string, probe func(string) (diskSample, error)) SpaceSnapshot {
	s := SpaceSnapshot{samples: map[filesystemID]diskSample{}}
	for _, path := range paths {
		if path == "" {
			continue
		}
		v, err := probe(path)
		if err != nil {
			if !os.IsNotExist(err) {
				s.warnings = append(s.warnings, fmt.Sprintf("Could not measure %s: %v", path, err))
			}
			continue
		}
		if _, exists := s.samples[v.ID]; !exists {
			s.samples[v.ID] = v
		}
	}
	return s
}

func (s SpaceSnapshot) Change() ([]FilesystemChange, []string) { return s.change(inspectDisk) }

func (s SpaceSnapshot) change(probe func(string) (diskSample, error)) ([]FilesystemChange, []string) {
	warnings := append([]string{}, s.warnings...)
	var changes []FilesystemChange
	for id, before := range s.samples {
		after, err := probe(before.Path)
		if err != nil || after.ID != id {
			warnings = append(warnings, "Space measurement unavailable or filesystem changed: "+before.Path)
			continue
		}
		changes = append(changes, FilesystemChange{Path: before.Path, Bytes: after.Available - before.Available})
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	sort.Strings(warnings)
	return changes, warnings
}
