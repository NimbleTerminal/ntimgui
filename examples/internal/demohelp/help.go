// Copyright (c) 2026 Neomantra Corp.

// Package demohelp connects ntimgui example help keys to native and browser booba controls.
package demohelp

import (
	"fmt"
	"os"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	booba "nimbleterminal.dev/booba"
	"nimbleterminal.dev/booba/actions"
	"nimbleterminal.dev/booba/actions/bubblehelp"
)

// Demo is implemented by interactive ntimgui examples hosted via booba.
type Demo interface {
	tea.Model
	Help() []key.Binding
	Summary() string
}

type helpKeys []key.Binding

func (k helpKeys) ShortHelp() []key.Binding  { return k }
func (k helpKeys) FullHelp() [][]key.Binding { return [][]key.Binding{k} }

// Key constructs a Bubbles key binding with its primary key shown in help/action bars.
func Key(label string, keys ...string) key.Binding {
	return key.NewBinding(key.WithKeys(keys...), key.WithHelp(keys[0], label))
}

// Quit reports whether msg is a standard quit shortcut (q or ctrl+c).
func Quit(msg tea.KeyPressMsg) bool {
	return msg.String() == "q" || msg.String() == "ctrl+c"
}

// Run wraps a Demo in bubblehelp action publication and executes it with booba.Run.
func Run(m Demo) error {
	wrapped, err := bubblehelp.NewModel(m, func(inner tea.Model) bubblehelp.Description {
		d := inner.(Demo)
		return bubblehelp.Description{
			Context: "ntimgui-demo",
			Keys:    helpKeys(d.Help()),
			Status:  actions.ProgramStatus{App: "ntimgui", Summary: d.Summary()},
		}
	}, bubblehelp.ModelOptions{
		Adapter: bubblehelp.Options{
			Customize: func(_ key.Binding, a *actions.Action) error {
				first := 0
				a.Preferred = &first
				return nil
			},
		},
		OnError: func(err error) { fmt.Fprintln(os.Stderr, err) },
	})
	if err != nil {
		return err
	}
	return booba.Run(wrapped)
}

// Main runs m via booba and exits non-zero on error.
func Main(m Demo) {
	if err := Run(m); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
