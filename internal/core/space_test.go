package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSpaceMeasurementDeduplicatesFilesystemsAndIncludesSeparateHome(t *testing.T) {
	before := map[string]diskSample{
		"/":          {ID: filesystemID{Device: 1}, Path: "/", Available: 1000},
		"/var/cache": {ID: filesystemID{Device: 1}, Path: "/var/cache", Available: 1000},
		"/home":      {ID: filesystemID{Device: 2}, Path: "/home", Available: 5000},
	}
	probe := func(path string) (diskSample, error) { return before[path], nil }
	s := sampleSpace([]string{"/", "/home", "/var/cache"}, probe)
	if len(s.samples) != 2 {
		t.Fatal("shared filesystem counted more than once")
	}
	before["/"] = diskSample{ID: filesystemID{Device: 1}, Path: "/", Available: 900}
	before["/home"] = diskSample{ID: filesystemID{Device: 2}, Path: "/home", Available: 8000}
	changes, warnings := s.change(probe)
	var total int64
	for _, c := range changes {
		total += c.Bytes
	}
	if total != 2900 || len(changes) != 2 || len(warnings) != 0 {
		t.Fatalf("incorrect delta: %+v %v", changes, warnings)
	}
}

func TestSpaceMeasurementOmitsChangedOrUnavailableMounts(t *testing.T) {
	s := sampleSpace([]string{"/fixture"}, func(path string) (diskSample, error) {
		return diskSample{ID: filesystemID{Device: 1}, Path: path, Available: 100}, nil
	})
	for _, failed := range []bool{false, true} {
		changes, warnings := s.change(func(path string) (diskSample, error) {
			if failed {
				return diskSample{}, errors.New("unavailable")
			}
			return diskSample{ID: filesystemID{Device: 2}, Path: path, Available: 999999}, nil
		})
		if len(changes) != 0 || len(warnings) != 1 {
			t.Fatalf("invalid delta reported: %+v %v", changes, warnings)
		}
	}
}

func TestRealFilesystemSamplesAreDeduplicatedWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	s := sampleSpace([]string{dir, child}, inspectDisk)
	if len(s.samples) != 1 || len(s.warnings) != 0 {
		t.Fatalf("filesystem not deduplicated: %+v", s)
	}
	changes, warnings := s.Change()
	if len(changes) != 1 || len(warnings) != 0 {
		t.Fatalf("measurement failed: %+v %v", changes, warnings)
	}
}
