package cli

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var spinnerFrames = []string{"|", "/", "-", "\\"}

// startSpinner renders a spinner with message while it runs, if and only if
// stdout is an interactive terminal and --json wasn't requested. It returns
// a stop function that clears the line; safe to call even when no spinner
// was started.
func startSpinner(cmd *cobra.Command, message string) (stop func()) {
	noop := func() {}
	if wantsJSON(cmd) {
		return noop
	}
	f, ok := cmd.OutOrStdout().(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return noop
	}

	var mu sync.Mutex
	done := make(chan struct{})
	stopped := make(chan struct{})

	go func() {
		defer close(stopped)
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		for i := 0; ; i++ {
			select {
			case <-done:
				return
			case <-ticker.C:
				mu.Lock()
				fmt.Fprintf(f, "\r%s %s", spinnerFrames[i%len(spinnerFrames)], message)
				mu.Unlock()
			}
		}
	}()

	return func() {
		close(done)
		<-stopped
		mu.Lock()
		fmt.Fprintf(f, "\r%s\r", strings.Repeat(" ", len(message)+2))
		mu.Unlock()
	}
}
