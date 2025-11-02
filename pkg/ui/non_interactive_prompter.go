
package ui

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/mattn/go-isatty"
)

type NonInteractivePrompter struct {
	stopChan    chan bool
	quoteFunc   func() (string, error)
}

func NewNonInteractivePrompter(quoteFunc func() (string, error)) *NonInteractivePrompter {
	return &NonInteractivePrompter{
		stopChan:    make(chan bool),
		quoteFunc:   quoteFunc,
	}
}

func (p *NonInteractivePrompter) Start() {
	if p.IsInteractive() {
		return // Don't start in interactive mode
	}

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
					return
				}
				c := color.New(color.FgGreen)
				c.Println(quote)
			}
		}
	}()
}

func (p *NonInteractivePrompter) Stop() {
	if isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return
	}
	p.stopChan <- true
}

func (p *NonInteractivePrompter) IsInteractive() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}
