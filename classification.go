package diffview

import "context"

// CommitBrief captures essential commit metadata for PR context.
type CommitBrief struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
}

// ClassificationInput is the complete input for story classification.
// It represents a PR's worth of changes: multiple commits with their combined diff.
type ClassificationInput struct {
	Repo    string        `json:"repo"`
	Branch  string        `json:"branch"`
	Commits []CommitBrief `json:"commits"`
	Diff    Diff          `json:"diff"`
}

// FirstCommitMessage returns the message of the first commit, or empty if none.
// This is a migration helper - prefer iterating Commits directly.
func (c ClassificationInput) FirstCommitMessage() string {
	if len(c.Commits) == 0 {
		return ""
	}
	return c.Commits[0].Message
}

// FirstCommitHash returns the hash of the first commit, or empty if none.
// This is a migration helper - prefer iterating Commits directly.
func (c ClassificationInput) FirstCommitHash() string {
	if len(c.Commits) == 0 {
		return ""
	}
	return c.Commits[0].Hash
}

// CaseID returns a unique identifier for this case using repo/branch format.
// This uniquely identifies a PR-level case for judgment linking.
func (c ClassificationInput) CaseID() string {
	return c.Repo + "/" + c.Branch
}

// StoryClassification is the LLM's structured output for a diff.
type StoryClassification struct {
	ChangeType string    `json:"change_type"` // bugfix, feature, refactor, chore, docs
	Narrative  string    `json:"narrative"`   // cause-effect, core-periphery, before-after, etc.
	Summary    string    `json:"summary"`     // One sentence describing the change
	Sections   []Section `json:"sections"`    // Ordered sections grouping related hunks
}

// Section groups related hunks with a narrative role.
type Section struct {
	Role        string    `json:"role"`        // problem, fix, test, core, supporting, etc.
	Title       string    `json:"title"`       // Human-readable section title
	Hunks       []HunkRef `json:"hunks"`       // References to hunks in this section
	Explanation string    `json:"explanation"` // Why this section matters
}

// HunkRef references a specific hunk with classification metadata.
type HunkRef struct {
	File         string `json:"file"`
	HunkIndex    int    `json:"hunk_index"`
	Category     string `json:"category"`                // refactoring, systematic, core, noise
	Collapsed    bool   `json:"collapsed"`               // Whether to collapse in viewer
	CollapseText string `json:"collapse_text,omitempty"` // Summary when collapsed
}

// OrderSections reorders sections based on the narrative type.
// This provides deterministic ordering regardless of LLM output order.
func (sc *StoryClassification) OrderSections() {
	if sc == nil || len(sc.Sections) == 0 {
		return
	}

	order := narrativeOrder(sc.Narrative)
	sortSectionsByOrder(sc.Sections, order)
}

// narrativeOrder returns the role ordering for a given narrative type.
// Most narratives end with supporting -> cleanup as a consistent suffix.
// Exception: before-after has cleanup at the start (it represents "before"/removal).
// Roles not in the returned slice are placed at the end in original order.
func narrativeOrder(narrative string) []string {
	switch narrative {
	case "cause-effect":
		return []string{"problem", "fix", "test", "supporting", "cleanup"}
	case "core-periphery":
		return []string{"core", "supporting", "cleanup"}
	case "before-after":
		return []string{"cleanup", "core", "test", "supporting"}
	case "rule-instances":
		return []string{"pattern", "core", "supporting", "cleanup"}
	case "entry-implementation":
		return []string{"interface", "core", "test", "supporting", "cleanup"}
	default:
		return nil
	}
}

// sortSectionsByOrder sorts sections in-place by the given role order.
// Roles not in the order slice maintain relative position at the end.
func sortSectionsByOrder(sections []Section, order []string) {
	if len(order) == 0 {
		return
	}

	// Build priority map: role -> position
	priority := make(map[string]int, len(order))
	for i, role := range order {
		priority[role] = i
	}
	unknownPriority := len(order) // Roles not in order go after known roles

	// Stable sort to preserve relative order of same-priority sections
	for i := 1; i < len(sections); i++ {
		for j := i; j > 0; j-- {
			pi := sectionPriority(sections[j-1], priority, unknownPriority)
			pj := sectionPriority(sections[j], priority, unknownPriority)
			if pi <= pj {
				break
			}
			sections[j-1], sections[j] = sections[j], sections[j-1]
		}
	}
}

// sectionPriority returns the sort priority for a section.
func sectionPriority(s Section, priority map[string]int, defaultPriority int) int {
	if p, ok := priority[s.Role]; ok {
		return p
	}
	return defaultPriority
}

// StoryClassifier produces structured classification from diff + commit info.
type StoryClassifier interface {
	Classify(ctx context.Context, input ClassificationInput) (*StoryClassification, error)
}
