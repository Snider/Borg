// Package ui provides components for creating command-line user interfaces,
// including progress bars and non-interactive prompters.
package ui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

// NonInteractivePrompter is used to display thematic quotes during long-running
// operations in non-interactive sessions (e.g., in a CI/CD pipeline). It can be
// started and stopped, and it will periodically print a quote from the provided
// quote function.
type NonInteractivePrompter struct {
	stopChan  chan struct{}
	quoteFunc func() (string, error)
	started   bool
	mu        sync.Mutex
	stopOnce  sync.Once
}

// NewNonInteractivePrompter creates a new NonInteractivePrompter with the given
// quote function.
//
// Example:
//
//	prompter := ui.NewNonInteractivePrompter(ui.GetVCSQuote)
//	prompter.Start()
//	// ... long-running operation ...
//	prompter.Stop()
func NewNonInteractivePrompter(quoteFunc func() (string, error)) *NonInteractivePrompter {
	return &NonInteractivePrompter{
		stopChan:  make(chan struct{}),
		quoteFunc: quoteFunc,
	}
}

// Start begins the prompter, which will periodically print quotes to the console
// in non-interactive sessions. It is safe to call Start multiple times.
func (p *NonInteractivePrompter) Start() {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return
	}
	p.started = true
	p.mu.Unlock()

	//if p.IsInteractive() {
	//	return // Don't start in interactive mode
	//}

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-p.stopChan:
				return
			case <-ticker.C:
				quote, err := p.quoteFunc()
				if err != nil {
					fmt.Println("Error getting quote:", err)
					continue
				}
				c := color.New(color.FgGreen)
				c.Println(quote)
			}
		}
	}()
}

// Stop halts the prompter. It is safe to call Stop multiple times.
func (p *NonInteractivePrompter) Stop() {
	if p.IsInteractive() {
		return
	}
	p.stopOnce.Do(func() {
		close(p.stopChan)
	})
}

// IsInteractive checks if the current session is interactive (i.e., running in
// a terminal).
func (p *NonInteractivePrompter) IsInteractive() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}
