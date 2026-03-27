package ui

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Spinner shows an animated spinner on stderr.
type Spinner struct {
	message string
	done    chan struct{}
	once    sync.Once
}

var frames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

// StartSpinner starts an animated spinner with a message.
func StartSpinner(message string) *Spinner {
	s := &Spinner{
		message: message,
		done:    make(chan struct{}),
	}

	go func() {
		i := 0
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.done:
				// Clear the spinner line
				fmt.Fprintf(os.Stderr, "\r\033[K")
				return
			case <-ticker.C:
				fmt.Fprintf(os.Stderr, "\r%s %s", frames[i%len(frames)], s.message)
				i++
			}
		}
	}()

	return s
}

// Stop stops the spinner.
func (s *Spinner) Stop() {
	s.once.Do(func() {
		close(s.done)
		// Small delay to let the goroutine clear the line
		time.Sleep(100 * time.Millisecond)
	})
}
