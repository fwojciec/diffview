package bubbletea_test

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fwojciec/diffview/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestDefaultKeyMap_HasExpectedBindings(t *testing.T) {
	t.Parallel()

	km := bubbletea.DefaultKeyMap()

	t.Run("Up binding", func(t *testing.T) {
		t.Parallel()
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		assert.True(t, key.Matches(msg, km.Up), "k should match Up binding")

		msg = tea.KeyMsg{Type: tea.KeyUp}
		assert.True(t, key.Matches(msg, km.Up), "arrow up should match Up binding")
	})

	t.Run("Down binding", func(t *testing.T) {
		t.Parallel()
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		assert.True(t, key.Matches(msg, km.Down), "j should match Down binding")

		msg = tea.KeyMsg{Type: tea.KeyDown}
		assert.True(t, key.Matches(msg, km.Down), "arrow down should match Down binding")
	})

	t.Run("HalfPageUp binding", func(t *testing.T) {
		t.Parallel()
		msg := tea.KeyMsg{Type: tea.KeyCtrlU}
		assert.True(t, key.Matches(msg, km.HalfPageUp), "ctrl+u should match HalfPageUp binding")
	})

	t.Run("HalfPageDown binding", func(t *testing.T) {
		t.Parallel()
		msg := tea.KeyMsg{Type: tea.KeyCtrlD}
		assert.True(t, key.Matches(msg, km.HalfPageDown), "ctrl+d should match HalfPageDown binding")
	})

	t.Run("GotoTop binding", func(t *testing.T) {
		t.Parallel()
		// Note: "gg" requires multi-key sequence handling in the Model
		// This test verifies that "g" is the trigger key
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		assert.True(t, key.Matches(msg, km.GotoTop), "g should match GotoTop binding")
	})

	t.Run("GotoBottom binding", func(t *testing.T) {
		t.Parallel()
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		assert.True(t, key.Matches(msg, km.GotoBottom), "G should match GotoBottom binding")
	})

	t.Run("Quit binding", func(t *testing.T) {
		t.Parallel()
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
		assert.True(t, key.Matches(msg, km.Quit), "q should match Quit binding")

		msg = tea.KeyMsg{Type: tea.KeyCtrlC}
		assert.True(t, key.Matches(msg, km.Quit), "ctrl+c should match Quit binding")
	})
}

func TestKeyMap_HelpText(t *testing.T) {
	t.Parallel()

	km := bubbletea.DefaultKeyMap()

	t.Run("bindings have help text", func(t *testing.T) {
		t.Parallel()

		// Verify help text is set for each binding
		assert.NotEmpty(t, km.Up.Help().Key, "Up should have help key")
		assert.NotEmpty(t, km.Up.Help().Desc, "Up should have help description")

		assert.NotEmpty(t, km.Down.Help().Key, "Down should have help key")
		assert.NotEmpty(t, km.Down.Help().Desc, "Down should have help description")

		assert.NotEmpty(t, km.Quit.Help().Key, "Quit should have help key")
		assert.NotEmpty(t, km.Quit.Help().Desc, "Quit should have help description")
	})
}
