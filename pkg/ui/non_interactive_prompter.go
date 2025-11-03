package ui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

// NonInteractivePrompter periodically prints status quotes when the terminal is non-interactive.
// It is intended to give the user feedback during long-running operations.
type NonInteractivePrompter struct {
	stopChan  chan struct{}
	quoteFunc func() (string, error)
	started   bool
	mu        sync.Mutex
	stopOnce  sync.Once
}

// NewNonInteractivePrompter returns a new NonInteractivePrompter that will call
// the provided quoteFunc to retrieve text to display.
func NewNonInteractivePrompter(quoteFunc func() (string, error)) *NonInteractivePrompter {
	return &NonInteractivePrompter{
		stopChan:  make(chan struct{}),
		quoteFunc: quoteFunc,
	}
}

// Start begins the periodic display of quotes until Stop is called.
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

// Stop signals the prompter to stop printing further messages.
func (p *NonInteractivePrompter) Stop() {
	if p.IsInteractive() {
		return
	}
	p.stopOnce.Do(func() {
		close(p.stopChan)
	})
}

// IsInteractive reports whether stdout is attached to an interactive terminal.
func (p *NonInteractivePrompter) IsInteractive() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}
