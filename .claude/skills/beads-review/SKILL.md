---
name: beads-review
description: Forward-looking code review that evaluates changes against downstream beads. Queries beads dependencies, evaluates structural discipline, and returns recommendations for issue updates.
---

# Beads-Aware Code Review

Reviews implementation for forward compatibility with downstream work. Primary output is **issue updates** that tighten specifications—not binary approval.

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

  ## Two Types of Findings

  ### A. Implementation Issues (require code changes)
  Actual bugs or structural violations IN THE CURRENT IMPLEMENTATION:
  - Hardcoded dependencies that should be injected
  - Violations of codebase patterns (e.g., Ben Johnson layout)
  - Assumptions that conflict with downstream requirements
  - Missing seams that downstream explicitly needs and current scope includes

  ### B. Specification Gaps (require issue updates)
  Opportunities or risks for downstream work that can be addressed by updating issue notes:
  - Wiring/integration steps downstream will need to perform
  - Architectural decisions that affect how downstream should approach its work
  - Ambiguity about where responsibilities lie between current and downstream tasks

  ## Key Distinction

  Ask: "Is this a problem with the implementation, or unclear scope?"

  - If the implementation is correct for its stated validation criteria → APPROVE + update downstream issues
  - If the implementation itself has structural violations → REJECT with specific fixes

  ## Output Format
  Return:
  1. VERDICT: APPROVE or REJECT
  2. IMPLEMENTATION_ISSUES: Problems in the code that require changes before merging (empty if none)
  3. DOWNSTREAM_UPDATES: Specific notes to add to downstream issues to ensure architectural coherence
  4. CURRENT_ISSUE_NOTES: Any clarifications to add to current issue (optional)

  Only REJECT for implementation issues. Specification gaps are handled via issue updates.
  """
)
```

### 3. Act on Results

**Always**: Apply DOWNSTREAM_UPDATES via `bd update <downstream-id> --notes "..."`

Based on verdict:
- **APPROVE**: Update downstream issues, proceed to finish task
- **REJECT**: Fix implementation issues, run `make validate`, re-review if changes were significant
