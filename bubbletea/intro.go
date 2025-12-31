package bubbletea

import (
	"github.com/charmbracelet/lipgloss"
	diffview "github.com/fwojciec/diffstory"
)

// NarrativeDiagram returns a visual representation of the story's narrative flow.
// The diagram adapts to the roles present in the sections.
// If renderer is nil, a default renderer is used.
func NarrativeDiagram(narrative string, sections []diffview.Section, renderer *lipgloss.Renderer) string {
	if len(sections) == 0 {
		return ""
	}
	// Use default renderer if nil (same pattern as newStyle in story.go)
	if renderer == nil {
		renderer = lipgloss.DefaultRenderer()
	}

	switch narrative {
	case "cause-effect", "entry-implementation", "before-after", "rule-instances":
		return linearFlowDiagram(sections, renderer)
	case "core-periphery":
		return hubAndSpokeDiagram(sections, renderer)
	default:
		return ""
	}
}

// linearFlowDiagram renders roles as a horizontal flow: role1 → role2 → role3
func linearFlowDiagram(sections []diffview.Section, renderer *lipgloss.Renderer) string {
	roles := extractRoles(sections)
	if len(roles) == 0 {
		return ""
	}

	nodeStyle := renderer.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	arrow := " → "

	// Pre-allocate: each role gets a node + arrow, minus trailing arrow
	parts := make([]string, 0, len(roles)*2-1)
	for _, role := range roles {
		parts = append(parts, nodeStyle.Render(role))
		parts = append(parts, arrow)
	}
	// Remove trailing arrow
	if len(parts) > 0 {
		parts = parts[:len(parts)-1]
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

// hubAndSpokeDiagram renders a hub-and-spoke diagram with core on the left.
// Peripheral roles are shown on the right, connected to the core box.
// This matches section order where core is typically first.
//
// Example output (with rounded borders):
//
//	╭──────╮── test
//	│ core │── supporting
//	╰──────╯── cleanup
func hubAndSpokeDiagram(sections []diffview.Section, renderer *lipgloss.Renderer) string {
	roles := extractRoles(sections)
	if len(roles) == 0 {
		return ""
	}

	// Find core role - it's the hub
	var hasCore bool
	var peripheralRoles []string
	for _, role := range roles {
		if role == "core" {
			hasCore = true
		} else {
			peripheralRoles = append(peripheralRoles, role)
		}
	}

	// No core = no hub-and-spoke diagram
	if !hasCore {
		return ""
	}

	nodeStyle := renderer.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)

	// Core is always centered
	coreNode := nodeStyle.Render("core")

	// If no peripheral roles, just show core
	if len(peripheralRoles) == 0 {
		return coreNode
	}

	// Build spoke connector
	spoke := "── "

	// Build peripheral roles column (right side)
	rightParts := make([]string, 0, len(peripheralRoles))
	for _, role := range peripheralRoles {
		rightParts = append(rightParts, spoke+role)
	}
	rightColumn := lipgloss.JoinVertical(lipgloss.Left, rightParts...)

	// Join core with right spokes
	return lipgloss.JoinHorizontal(lipgloss.Center, coreNode, rightColumn)
}

// extractRoles returns unique roles from sections in order.
func extractRoles(sections []diffview.Section) []string {
	var roles []string
	seen := make(map[string]bool)
	for _, s := range sections {
		if s.Role != "" && !seen[s.Role] {
			roles = append(roles, s.Role)
			seen[s.Role] = true
		}
	}
	return roles
}
