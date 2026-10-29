package app

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type readerFunc func([]byte) (int, error)

func (f readerFunc) Read(p []byte) (int, error) { return f(p) }

func TestConfirmationCanBeInterruptedWhileInputIsBlocked(t *testing.T) {
	r, w := io.Pipe()
	t.Cleanup(func() { r.Close(); w.Close() })
	started := make(chan struct{})
	reader := readerFunc(func(p []byte) (int, error) { close(started); return r.Read(p) })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		confirmed, err := confirm(ctx, reader)
		if confirmed {
			done <- errors.New("interrupted input confirmed cleanup")
			return
		}
		done <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal("unexpected confirmation result", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("confirmation ignored cancellation")
	}
}

func TestConfirmationNeedsACompleteAffirmativeLine(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  bool
	}{
		{"y\n", true}, {" YES \n", true}, {"n\n", false}, {"\n", false}, {"", false}, {"yes", false},
	} {
		got, err := confirm(context.Background(), strings.NewReader(tc.input))
		if got != tc.want || err != nil {
			t.Fatalf("%q: %v %v", tc.input, got, err)
		}
	}
}
