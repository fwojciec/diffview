---
name: beads-review
description: Forward-looking code review that evaluates changes against downstream beads. Queries beads dependencies, evaluates structural discipline, and returns recommendations for issue updates.
---

# Beads-Aware Code Review

## Workflow

### 1. Gather Context

```bash
ISSUE_ID=$(git branch --show-current)
bd show $ISSUE_ID
bd list --blocked-by $ISSUE_ID 2>/dev/null
# For each downstream issue:
bd show <downstream-id>
```

### 2. Launch Review Subagent

Use the Task tool with model: "sonnet":

```
Task(
  subagent_type: "general-purpose",
  model: "sonnet",
  prompt: """
  Review this implementation for forward compatibility.

  ## Current Issue
  [bd show output]

  ## Downstream Dependencies
  [downstream issue descriptions, or "None"]

  ## Diff
  [git diff main...HEAD]

  ## Structural Checklist
  - [ ] Dependencies injected, not hardcoded
  - [ ] Single responsibility maintained
  - [ ] Seams present for testing
  - [ ] Interface boundaries clean
  - [ ] Consistent with codebase patterns
  - [ ] No assumptions conflicting with downstream work

  ## YAGNI Distinction
  - Feature YAGNI: Don't build features you don't need yet. Reject suggestions to add speculative functionality.
  - Structural YAGNI: DO maintain architectural discipline even when "you don't need it yet." Accept suggestions that add seams, boundaries, or dependency injection.

  The question: "Does this preserve our capacity for change?" - not "Do we need this feature?"

  ## Output Format
  Return:
  1. VERDICT: APPROVE or REJECT
  2. STRUCTURAL_FINDINGS: List any violations with specific file:line references
  3. DOWNSTREAM_FRICTION: For each downstream issue, note any friction this creates
  4. RECOMMENDED_ISSUE_UPDATES: Suggested notes to add to current or downstream issues
  5. REJECTION_FEEDBACK: If rejecting, specific changes needed

  Bias toward rejection when structural discipline is compromised.
  """
)
```

### 3. Act on Results

Based on subagent response:
- **APPROVE**: Proceed to finish task
- **REJECT**: Apply feedback, return to implementation

Use RECOMMENDED_ISSUE_UPDATES to inform `bd update` commands.
