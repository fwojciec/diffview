package diffview

import "encoding/json"

// AnnotatedHunk wraps a Hunk with an ID for LLM reference.
type AnnotatedHunk struct {
	ID   string // Unique identifier for referencing in analysis
	Hunk Hunk
}

// DiffAnalysis is an extensible container for diff analyses.
type DiffAnalysis struct {
	Version  int        // Schema version for forward compatibility
	Analyses []Analysis // Multiple analysis types can be included
}

// Analysis represents a single analysis result with a type discriminator.
type Analysis struct {
	Type    string          // e.g., "story", "security", "complexity"
	Payload json.RawMessage // Type-specific JSON payload
}

// StoryAnalysis describes the narrative structure of a code change.
type StoryAnalysis struct {
	ChangeType string      // e.g., "refactor", "feature", "bugfix"
	Summary    string      // One-line description of what changed
	Parts      []StoryPart // Ordered sequence of change components
}

// StoryPart represents one component of a change story.
type StoryPart struct {
	Role        string   // e.g., "setup", "core", "cleanup"
	HunkIDs     []string // References to AnnotatedHunk IDs
	Explanation string   // Human-readable description of this part
}
