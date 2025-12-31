package bubbletea

import (
	"github.com/charmbracelet/lipgloss"
	diffview "github.com/fwojciec/diffstory"
)

// NarrativeDiagram returns a visual representation of the story's narrative flow.
// The diagram adapts to the roles present in the sections.
func NarrativeDiagram(narrative string, sections []diffview.Section, renderer *lipgloss.Renderer) string {
	if len(sections) == 0 || renderer == nil {
		return ""
	}

	switch narrative {
	case "cause-effect", "entry-implementation", "before-after", "rule-instances":
		return linearFlowDiagram(sections, renderer)
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
