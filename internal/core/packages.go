package core

import (
	"regexp"
	"strings"
)

type SnapRevision struct {
	Name     string `json:"name"`
	Revision string `json:"revision"`
	Disabled bool   `json:"disabled"`
}

var snapName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
var snapRevision = regexp.MustCompile(`^x?[0-9]+$`)

func ParseSnaps(output string) []SnapRevision {
	var result []SnapRevision
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 || !snapName.MatchString(fields[0]) || !snapRevision.MatchString(fields[2]) {
			continue
		}
		disabled := false
		for _, note := range strings.Split(fields[len(fields)-1], ",") {
			if note == "disabled" {
				disabled = true
			}
		}
		result = append(result, SnapRevision{Name: fields[0], Revision: fields[2], Disabled: disabled})
	}
	return result
}
