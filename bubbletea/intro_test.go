package bubbletea_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	diffview "github.com/fwojciec/diffstory"
	"github.com/fwojciec/diffstory/bubbletea"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

func TestNarrativeDiagram_CauseEffect(t *testing.T) {
	t.Parallel()

	// cause-effect narrative with problem, fix, test roles
	sections := []diffview.Section{
		{Role: "problem", Title: "The Bug"},
		{Role: "fix", Title: "The Solution"},
		{Role: "test", Title: "Verification"},
	}

	renderer := lipgloss.NewRenderer(nil, termenv.WithProfile(termenv.Ascii))
	diagram := bubbletea.NarrativeDiagram("cause-effect", sections, renderer)

	// Should show linear flow with arrows between roles
	assert.Contains(t, diagram, "problem")
	assert.Contains(t, diagram, "fix")
	assert.Contains(t, diagram, "test")
	assert.Contains(t, diagram, "→")
}

func TestNarrativeDiagram_EntryImplementation(t *testing.T) {
	t.Parallel()

	// entry-implementation narrative with entry and implementation roles
	sections := []diffview.Section{
		{Role: "entry", Title: "API Contract"},
		{Role: "implementation", Title: "Core Logic"},
	}

	renderer := lipgloss.NewRenderer(nil, termenv.WithProfile(termenv.Ascii))
	diagram := bubbletea.NarrativeDiagram("entry-implementation", sections, renderer)

	// Should show linear flow with entry and implementation roles
	assert.Contains(t, diagram, "entry")
	assert.Contains(t, diagram, "implementation")
	assert.Contains(t, diagram, "→")
}

func TestNarrativeDiagram_BeforeAfter(t *testing.T) {
	t.Parallel()

	// before-after narrative with cleanup and core roles
	sections := []diffview.Section{
		{Role: "cleanup", Title: "Remove old code"},
		{Role: "core", Title: "Add new code"},
	}

	renderer := lipgloss.NewRenderer(nil, termenv.WithProfile(termenv.Ascii))
	diagram := bubbletea.NarrativeDiagram("before-after", sections, renderer)

	// Should show transformation flow
	assert.Contains(t, diagram, "cleanup")
	assert.Contains(t, diagram, "core")
	assert.Contains(t, diagram, "→")
}

func TestNarrativeDiagram_EmptySections(t *testing.T) {
	t.Parallel()

	renderer := lipgloss.NewRenderer(nil, termenv.WithProfile(termenv.Ascii))
	diagram := bubbletea.NarrativeDiagram("cause-effect", nil, renderer)

	assert.Empty(t, diagram)
}

func TestNarrativeDiagram_NoRoles(t *testing.T) {
	t.Parallel()

	// Sections without roles
	sections := []diffview.Section{
		{Title: "First"},
		{Title: "Second"},
	}

	renderer := lipgloss.NewRenderer(nil, termenv.WithProfile(termenv.Ascii))
	diagram := bubbletea.NarrativeDiagram("cause-effect", sections, renderer)

	assert.Empty(t, diagram)
}

func TestNarrativeDiagram_UnknownNarrative(t *testing.T) {
	t.Parallel()

	sections := []diffview.Section{
		{Role: "core", Title: "Changes"},
	}

	renderer := lipgloss.NewRenderer(nil, termenv.WithProfile(termenv.Ascii))
	diagram := bubbletea.NarrativeDiagram("unknown-narrative", sections, renderer)

	assert.Empty(t, diagram)
}

func TestNarrativeDiagram_DeduplicatesRoles(t *testing.T) {
	t.Parallel()

	// Multiple sections with same role should show role only once
	sections := []diffview.Section{
		{Role: "fix", Title: "First fix"},
		{Role: "fix", Title: "Second fix"},
		{Role: "test", Title: "Tests"},
	}

	renderer := lipgloss.NewRenderer(nil, termenv.WithProfile(termenv.Ascii))
	diagram := bubbletea.NarrativeDiagram("cause-effect", sections, renderer)

	// Count occurrences of "fix" - should appear only once in diagram
	// The diagram contains borders, so we check the role appears in a box
	assert.Contains(t, diagram, "fix")
	assert.Contains(t, diagram, "test")
	// There should be exactly one arrow (between fix and test)
	count := strings.Count(diagram, "→")
	assert.Equal(t, 1, count, "should have exactly one arrow between two unique roles")
}

func TestNarrativeDiagram_RuleInstances(t *testing.T) {
	t.Parallel()

	// rule-instances narrative with pattern and instance roles
	sections := []diffview.Section{
		{Role: "pattern", Title: "The Pattern"},
		{Role: "instance", Title: "First Application"},
		{Role: "instance", Title: "Second Application"},
	}

	renderer := lipgloss.NewRenderer(nil, termenv.WithProfile(termenv.Ascii))
	diagram := bubbletea.NarrativeDiagram("rule-instances", sections, renderer)

	// Should show flow with pattern and instance
	assert.Contains(t, diagram, "pattern")
	assert.Contains(t, diagram, "instance")
	assert.Contains(t, diagram, "→")
}

func TestNarrativeDiagram_CorePeriphery(t *testing.T) {
	t.Parallel()

	// core-periphery narrative should produce a hub-and-spoke diagram
	// with core in the center and other roles radiating from it
	sections := []diffview.Section{
		{Role: "core", Title: "Main Change"},
		{Role: "supporting", Title: "Ripple Effect"},
		{Role: "test", Title: "Verification"},
	}

	renderer := lipgloss.NewRenderer(nil, termenv.WithProfile(termenv.Ascii))
	diagram := bubbletea.NarrativeDiagram("core-periphery", sections, renderer)

	// Hub-and-spoke should show core with connections to other roles
	assert.Contains(t, diagram, "core")
	assert.Contains(t, diagram, "supporting")
	assert.Contains(t, diagram, "test")
	// Should NOT have linear arrows (that's for linear narratives)
	assert.NotContains(t, diagram, "→")
}

func TestNarrativeDiagram_CorePeriphery_SingleRole(t *testing.T) {
	t.Parallel()

	// With only core role, should still render (even if no spokes)
	sections := []diffview.Section{
		{Role: "core", Title: "Only Core"},
	}

	renderer := lipgloss.NewRenderer(nil, termenv.WithProfile(termenv.Ascii))
	diagram := bubbletea.NarrativeDiagram("core-periphery", sections, renderer)

	assert.Contains(t, diagram, "core")
}

func TestNarrativeDiagram_CorePeriphery_NoCore(t *testing.T) {
	t.Parallel()

	// Without core role, should return empty (can't have hub without hub)
	sections := []diffview.Section{
		{Role: "supporting", Title: "Support Only"},
		{Role: "test", Title: "Test Only"},
	}

	renderer := lipgloss.NewRenderer(nil, termenv.WithProfile(termenv.Ascii))
	diagram := bubbletea.NarrativeDiagram("core-periphery", sections, renderer)

	// No core = no hub-and-spoke diagram
	assert.Empty(t, diagram)
}

func TestNarrativeDiagram_CorePeriphery_ManyRoles(t *testing.T) {
	t.Parallel()

	// With multiple peripheral roles, all should be shown
	sections := []diffview.Section{
		{Role: "core", Title: "Main"},
		{Role: "supporting", Title: "Support 1"},
		{Role: "supporting", Title: "Support 2"},
		{Role: "test", Title: "Tests"},
		{Role: "cleanup", Title: "Cleanup"},
	}

	renderer := lipgloss.NewRenderer(nil, termenv.WithProfile(termenv.Ascii))
	diagram := bubbletea.NarrativeDiagram("core-periphery", sections, renderer)

	assert.Contains(t, diagram, "core")
	assert.Contains(t, diagram, "supporting")
	assert.Contains(t, diagram, "test")
	assert.Contains(t, diagram, "cleanup")
	// Roles should be deduplicated
	count := strings.Count(diagram, "supporting")
	assert.Equal(t, 1, count, "supporting should appear only once")
}

func TestNarrativeDiagram_NilRenderer(t *testing.T) {
	t.Parallel()

	sections := []diffview.Section{
		{Role: "fix", Title: "The Fix"},
	}

	diagram := bubbletea.NarrativeDiagram("cause-effect", sections, nil)

	// Should use default renderer when nil is passed
	assert.Contains(t, diagram, "fix")
}

func TestNarrativeDiagram_CorePeriphery_CoreAppearsFirst(t *testing.T) {
	t.Parallel()

	// Core is section 1, so it should appear on the LEFT (first in reading order)
	// to match the section sequence: 1. core, 2. test, 3. supporting
	sections := []diffview.Section{
		{Role: "core", Title: "Main Change"},
		{Role: "test", Title: "Verification"},
		{Role: "supporting", Title: "Ripple Effect"},
	}

	renderer := lipgloss.NewRenderer(nil, termenv.WithProfile(termenv.Ascii))
	diagram := bubbletea.NarrativeDiagram("core-periphery", sections, renderer)

	// Core should appear before peripheral roles in the string
	// (i.e., core is on the left side of the diagram)
	corePos := strings.Index(diagram, "core")
	testPos := strings.Index(diagram, "test")
	supportingPos := strings.Index(diagram, "supporting")

	assert.Less(t, corePos, testPos, "core should appear before test in diagram")
	assert.Less(t, corePos, supportingPos, "core should appear before supporting in diagram")
}
