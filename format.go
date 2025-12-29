package diffview

import (
	"fmt"
	"strings"
)

// PromptFormatter renders classification input as structured text for LLM prompts.
type PromptFormatter interface {
	Format(input ClassificationInput) string
}

// DefaultFormatter implements PromptFormatter with the standard format.
type DefaultFormatter struct{}

// Format renders the classification input as structured text.
func (f *DefaultFormatter) Format(input ClassificationInput) string {
	var sb strings.Builder

	// Context section with PR metadata
	sb.WriteString("<context>\n")
	sb.WriteString(fmt.Sprintf("Repository: %s\n", input.Repo))
	if input.Branch != "" {
		sb.WriteString(fmt.Sprintf("Branch: %s\n", input.Branch))
	}
	if input.PRTitle != "" {
		sb.WriteString(fmt.Sprintf("PR Title: %s\n", input.PRTitle))
	}
	if input.PRDescription != "" {
		sb.WriteString(fmt.Sprintf("PR Description:\n%s\n", input.PRDescription))
	}
	if len(input.Commits) > 0 {
		sb.WriteString("\nCommits:\n")
		for _, c := range input.Commits {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", c.Hash, c.Message))
		}
	}
	sb.WriteString("</context>\n\n")

	// Diff section
	sb.WriteString("<diff>\n")

	hunkNum := 1
	for _, file := range input.Diff.Files {
		// File header
		sb.WriteString(fmt.Sprintf("=== FILE: %s (%s) ===\n\n",
			filePath(file), operationName(file.Operation)))

		// Hunks
		for _, hunk := range file.Hunks {
			sb.WriteString(fmt.Sprintf("--- HUNK H%d (@@ -%d,%d +%d,%d @@) ---\n",
				hunkNum, hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount))
			for _, line := range hunk.Lines {
				prefix := linePrefix(line.Type)
				sb.WriteString(prefix)
				sb.WriteString(line.Content)
				if !strings.HasSuffix(line.Content, "\n") {
					sb.WriteString("\n")
				}
			}
			sb.WriteString("\n")
			hunkNum++
		}
	}

	sb.WriteString("</diff>")
	return sb.String()
}

func filePath(file FileDiff) string {
	if file.NewPath != "" {
		return file.NewPath
	}
	return file.OldPath
}

func operationName(op FileOp) string {
	switch op {
	case FileAdded:
		return "added"
	case FileDeleted:
		return "deleted"
	case FileModified:
		return "modified"
	case FileRenamed:
		return "renamed"
	case FileCopied:
		return "copied"
	default:
		return "modified"
	}
}

func linePrefix(lt LineType) string {
	switch lt {
	case LineAdded:
		return "+"
	case LineDeleted:
		return "-"
	default:
		return " "
	}
}
