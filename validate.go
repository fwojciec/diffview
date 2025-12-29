package diffview

import "fmt"

// ValidationReason identifies why a HunkRef is invalid.
type ValidationReason string

// Validation error reasons.
const (
	ErrInvalidHunkIndex ValidationReason = "invalid_index"
	ErrFileNotFound     ValidationReason = "file_not_found"
)

// ValidationError describes a single validation failure in a classification.
type ValidationError struct {
	Section   int              // Index of the section containing the error
	HunkRef   HunkRef          // The problematic hunk reference
	Reason    ValidationReason // Why this reference is invalid
	HunkCount int              // Actual hunk count for the file (for invalid_index errors)
}

// Error implements the error interface.
func (e ValidationError) Error() string {
	switch e.Reason {
	case ErrInvalidHunkIndex:
		maxIndex := e.HunkCount - 1
		if maxIndex < 0 {
			return fmt.Sprintf("section %d: file %q has no hunks, hunk_index %d is invalid",
				e.Section, e.HunkRef.File, e.HunkRef.HunkIndex)
		}
		return fmt.Sprintf("section %d: file %q hunk_index %d is out of bounds (valid: 0-%d)",
			e.Section, e.HunkRef.File, e.HunkRef.HunkIndex, maxIndex)
	case ErrFileNotFound:
		return fmt.Sprintf("section %d: file %q not found in diff",
			e.Section, e.HunkRef.File)
	default:
		return fmt.Sprintf("section %d: unknown error for file %q hunk_index %d",
			e.Section, e.HunkRef.File, e.HunkRef.HunkIndex)
	}
}

// ValidateClassification checks that all hunk references in a classification
// are valid for the given diff. Returns a slice of validation errors, or nil
// if the classification is valid.
func ValidateClassification(diff *Diff, classification *StoryClassification) []ValidationError {
	// Build a map of file paths to their hunk counts for fast lookup
	hunkCounts := make(map[string]int)
	for _, file := range diff.Files {
		// Use NewPath as the canonical path (handles renames/additions)
		path := file.NewPath
		if path == "" {
			path = file.OldPath // For deletions
		}
		if path == "" {
			continue // Skip malformed file entries
		}
		hunkCounts[path] = len(file.Hunks)
	}

	var errors []ValidationError

	for sectionIdx, section := range classification.Sections {
		for _, ref := range section.Hunks {
			count, found := hunkCounts[ref.File]
			if !found {
				errors = append(errors, ValidationError{
					Section: sectionIdx,
					HunkRef: ref,
					Reason:  ErrFileNotFound,
				})
				continue
			}

			if ref.HunkIndex < 0 || ref.HunkIndex >= count {
				errors = append(errors, ValidationError{
					Section:   sectionIdx,
					HunkRef:   ref,
					Reason:    ErrInvalidHunkIndex,
					HunkCount: count,
				})
			}
		}
	}

	return errors
}
