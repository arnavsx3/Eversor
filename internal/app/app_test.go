package app

import (
	"bytes"
	"context"
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
