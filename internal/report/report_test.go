package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/arnavsx3/Eversor/internal/core"
)

func TestSummaryReportsSeparateFilesystemsAndUnavailableMeasurement(t *testing.T) {
	var out bytes.Buffer
	Summary(&out, core.Report{
		FreedBytes:          2900,
		Filesystems:         []core.FilesystemChange{{Path: "/", Bytes: -100}, {Path: "/home", Bytes: 3000}},
		MeasurementWarnings: []string{"Space measurement unavailable: /removed"},
	})
	for _, expected := range []string{"2.9 KB", "/: -100.0 B", "/home: 3.0 KB", "Measurement note:", "/removed"} {
		if !strings.Contains(out.String(), expected) {
			t.Fatalf("missing %q in %s", expected, out.String())
		}
	}
}
