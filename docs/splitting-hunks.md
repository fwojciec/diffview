# LLM-Defined Semantic Segmentation for Code Diffs

**LLMs can semantically segment code diffs, but require careful format design to overcome systematic line-number inaccuracies.** Research shows GPT-4 achieves only ~40-48% accuracy on identifying specific "cause lines" in code, and models struggle fundamentally with line counting because they must "manually parse and number every line" rather than using tokenization. The solution combines redundant coordinate specification, explicit old/new file disambiguation, validation feedback loops, and constraining outputs to proven reliable formats.

## Markup format design: combining JSON structure with redundant verification

The optimal format for LLM-specified code segments balances **parseability**, **LLM reliability**, and **diff coordinate semantics**. After analyzing GitHub API, Gerrit API, LSP protocol, and LLM structured output research, the recommended format is:

```json
{
  "segments": [{
    "id": "seg-1",
    "description": "Extract validation logic to helper function",
    "ranges": [{
      "path": "pkg/validator.go",
      "side": "new",
      "start_line": 42,
      "end_line": 67,
      "anchor_text": "func validateInput(ctx context.Context"
    }],
    "group_id": "refactor-validation"
  }]
}
```

**Why this format over alternatives:**

| Format | Parseability | LLM Reliability | Diff Semantics |
|--------|-------------|-----------------|----------------|
| `file.go:10-25` | High | Medium (off-by-one errors) | Ambiguous old/new |
| JSON with `side` field | High | Higher with anchor_text | Explicit disambiguation |
| LSP Position/Range | Very High | Lower (zero-based confuses LLMs) | Single-file only |
| GitHub API position | Medium | Low (deprecated counting system) | Hunk-relative |

The `anchor_text` field is critical—it provides **redundant verification** that catches line-number drift. Research on LLM code tasks found that including both line numbers AND content verification significantly reduces errors. The `side` field explicitly disambiguates old-file versus new-file coordinates, addressing unified diff's inherent ambiguity where a single position can reference two different line numbers.

**Handling edge cases** requires explicit status markers:

```json
{
  "path": "old_name.go",
  "status": "renamed",
  "new_path": "new_name.go",
  "side": "new",
  "start_line": 1,
  "end_line": 45
}
```

For **deleted files**, segments must use `"side": "old"` since new-file lines don't exist. For **added files**, only `"side": "new"` is valid. Binary files should be marked `"binary": true` with file-level-only references (no line ranges).

## Cross-file grouping: hierarchical segments within logical change units

Current tooling reveals a significant gap: **only GitHub Copilot Change Groups (2025) offers automatic semantic grouping**, while academic research shows 70%+ of semantically related classes will co-change. The recommended architecture uses a two-level hierarchy:

```json
{
  "groups": [{
    "id": "refactor-validation",
    "description": "Extract and centralize input validation",
    "rationale": "Reduces duplication across 3 handlers",
    "segments": ["seg-1", "seg-2", "seg-3"]
  }],
  "segments": [...]
}
```

**Groups span files; segments stay within files.** This mirrors how Microsoft's ClusterChanges research found developers naturally partition changes—using **def-use relationships** as the organizing principle, where uses of types, methods, or fields cluster with their definitions.

Cross-file grouping should leverage these research-backed heuristics:
- **Structural coupling**: Files importing/calling each other belong together
- **Semantic coupling**: Files sharing domain terminology (70%+ co-change rate)
- **Logical coupling**: Files with historical co-change patterns from version control
- **Refactoring patterns**: Rename propagations, extract interface changes span multiple files predictably

The key tradeoff: **tight grouping aids comprehension but complicates coverage validation**. Groups that span files require tracking which lines in which files belong to each group, adding complexity. The hierarchy approach keeps segments file-local (easier to validate) while groups provide semantic context (easier to understand).

## Granularity heuristics grounded in empirical research

The SmartBear/Cisco study of 2,500 code reviews (3.2M lines) established definitive thresholds:

| Metric | Optimal Range | Degraded Performance |
|--------|--------------|---------------------|
| Lines per review | **200-400 LOC** | >400 LOC: defect detection drops sharply |
| Review duration | **60-90 minutes** | >60 min: effectiveness plummets |
| Inspection rate | **300-500 LOC/hour** | >500 LOC/hr: superficial review |
| Expected yield | **70-90% defects found** | — |

**Concrete segmentation rules derived from research:**

1. **Primary boundary: method/function level**. Microsoft's ClusterChanges found 80% developer agreement when partitioning at method boundaries using def-use relationships.

2. **Maximum segment size: 50 lines of changed code** (not total lines). This keeps individual segments reviewable in ~10 minutes, matching working memory constraints (~4 chunks) and attention span research.

3. **Minimum segment size: 3 lines**. Single-line segments fragment context unnecessarily. Exception: security-critical single-line changes warrant isolation.

4. **Split triggers:**
   - Different logical concerns (bug fix vs. feature vs. refactoring)
   - Different system boundaries (API layer vs. database layer)
   - Unrelated def-use clusters
   - >50 changed lines in one logical unit

5. **Merge triggers:**
   - Related def-use chains <50 lines combined
   - Rename propagations affecting multiple locations
   - Test + implementation pairs for same feature

The "10 lines of code" guideline circulating in developer culture is **not research-backed**—it's an ironic observation that tiny PRs get scrutinized while massive PRs get rubber-stamped. The evidence supports 200-400 lines total, with segments of 20-50 changed lines each.

## Coverage validation algorithm using interval trees

Validating that segments cover a diff exactly—no gaps, no overlaps—requires efficient range operations. The algorithm uses an **interval tree** per file:

```python
from dataclasses import dataclass
from typing import List, Tuple, Optional
from sortedcontainers import SortedList

@dataclass
class LineRange:
    start: int  # inclusive, 1-based
    end: int    # inclusive, 1-based
    segment_id: str
    
class CoverageValidator:
    def __init__(self, diff_ranges: dict[str, List[Tuple[int, int]]]):
        """diff_ranges: {filepath: [(start, end), ...]} of changed lines"""
        self.diff_ranges = diff_ranges
        self.segment_ranges: dict[str, List[LineRange]] = {}
    
    def add_segment(self, path: str, start: int, end: int, segment_id: str):
        if path not in self.segment_ranges:
            self.segment_ranges[path] = []
        self.segment_ranges[path].append(LineRange(start, end, segment_id))
    
    def validate(self) -> dict:
        errors = {"gaps": [], "overlaps": [], "out_of_bounds": [], "missing_files": []}
        
        # Check all diff files are covered
        for path in self.diff_ranges:
            if path not in self.segment_ranges:
                errors["missing_files"].append(path)
                continue
            
            segments = sorted(self.segment_ranges[path], key=lambda r: r.start)
            diff_lines = set()
            for start, end in self.diff_ranges[path]:
                diff_lines.update(range(start, end + 1))
            
            covered_lines = set()
            prev_end = 0
            
            for seg in segments:
                # Check for overlaps with previous segment
                if seg.start <= prev_end:
                    errors["overlaps"].append({
                        "path": path,
                        "segment": seg.segment_id,
                        "overlap_at": seg.start,
                        "message": f"Line {seg.start} already covered"
                    })
                
                # Check for out-of-bounds (covering non-changed lines)
                seg_lines = set(range(seg.start, seg.end + 1))
                out_of_bounds = seg_lines - diff_lines
                if out_of_bounds:
                    errors["out_of_bounds"].append({
                        "path": path,
                        "segment": seg.segment_id,
                        "lines": sorted(out_of_bounds)[:5],  # First 5 for brevity
                        "message": f"Segment covers unchanged lines"
                    })
                
                covered_lines.update(seg_lines & diff_lines)
                prev_end = seg.end
            
            # Check for gaps
            gaps = diff_lines - covered_lines
            if gaps:
                gap_ranges = self._lines_to_ranges(sorted(gaps))
                errors["gaps"].append({
                    "path": path,
                    "uncovered_ranges": gap_ranges,
                    "message": f"Lines {gap_ranges[0]} not covered by any segment"
                })
        
        return errors
    
    def _lines_to_ranges(self, lines: List[int]) -> List[str]:
        """Convert [1,2,3,7,8] to ['1-3', '7-8']"""
        if not lines:
            return []
        ranges = []
        start = prev = lines[0]
        for line in lines[1:]:
            if line != prev + 1:
                ranges.append(f"{start}-{prev}" if start != prev else str(start))
                start = line
            prev = line
        ranges.append(f"{start}-{prev}" if start != prev else str(start))
        return ranges

    def format_llm_feedback(self, errors: dict) -> str:
        """Generate actionable feedback for LLM self-correction"""
        if not any(errors.values()):
            return "Coverage validation passed: all changed lines are covered exactly once."
        
        feedback = ["VALIDATION FAILED. Please correct these issues:\n"]
        
        for gap in errors["gaps"]:
            feedback.append(
                f"GAP: {gap['path']} has uncovered changed lines at {gap['uncovered_ranges']}. "
                f"Add a segment covering these lines."
            )
        
        for overlap in errors["overlaps"]:
            feedback.append(
                f"OVERLAP: {overlap['path']} line {overlap['overlap_at']} is covered by "
                f"multiple segments. Adjust {overlap['segment']} to avoid overlap."
            )
        
        for oob in errors["out_of_bounds"]:
            feedback.append(
                f"OUT OF BOUNDS: {oob['path']} segment {oob['segment']} covers unchanged "
                f"lines {oob['lines']}. Reduce range to only changed lines."
            )
        
        return "\n".join(feedback)
```

**Line range semantics**: Use **1-based inclusive** bounds (`start=10, end=20` means lines 10 through 20). This matches human mental models and unified diff conventions. Zero-based exclusive ranges (LSP-style) cause more LLM errors because models are trained on human-written content using 1-based counting.

**Context line handling**: Unchanged context lines in diffs should be **excluded** from segment ranges. Segments cover only lines prefixed with `+` or `-` in unified diff. To compute valid ranges, first extract changed line numbers from diff hunks, then validate segments against only those lines.

## Prior art: semantic diff tools and LLM integration research

**Semantic diff algorithms:**

- **GumTree** (Falleri et al., 2014): AST-based differencing with O(n²) complexity. Two-phase algorithm: top-down subtree matching, then bottom-up recovery. Detects moves and renames, not just insertions/deletions. Supports Java, JavaScript, Python, C via tree-sitter and language-specific parsers.

- **difftastic**: Treats diffing as a graph problem using Dijkstra's algorithm. Written in Rust, supports 60+ languages via tree-sitter. Ignores formatting changes, shows actual line numbers. Limitation: scales poorly on files with many changes.

- **Change Distilling** (Fluri et al., 2007): Fine-grained source code change extraction using improved tree differencing. Achieved 34% mean absolute percentage error versus 79% for baseline Chawathe algorithm.

**Academic research on code review cognition:**

- Bacchelli & Bird (2013), "Expectations, Outcomes, and Challenges of Modern Code Review": Found understanding is the key challenge—"the most difficult thing is understanding the reason of the change."

- Microsoft ClusterChanges (Barnett et al., 2015): Over 40% of changesets could be decomposed into independent partitions; def-use relationships are the primary organizing principle.

- Semantic coupling research (Springer, 2017): 70%+ of semantically related classes co-change; semantic coupling "better estimates the mental model of developers than other coupling measures."

**LLM + diff analysis tools:**

- **CodeRabbit**: Multi-model recursive reviews (o4-mini, o3, GPT-4.1). 46% accuracy detecting runtime bugs. Uses AST analysis via tree-sitter, enriches diff with code history and linter output.

- **Qodo PR-Agent** (open source): Single LLM call per tool with PR Compression strategy. Industrial deployment at Beko showed 73.8% of automated comments were acted upon.

- **Self-Refine framework**: Iterative feedback loop achieving 21-32 percentage point improvements in code task accuracy through external validation (linting, execution, tests).

## Pitfalls and mitigation strategies for LLM line references

**Systematic LLM failures with coordinates:**

| Pitfall | Frequency | Mitigation |
|---------|-----------|------------|
| Off-by-one errors | Very common | Require `anchor_text` verification field |
| Zero-based vs 1-based confusion | Common | Always use 1-based in prompts; explicit instruction |
| Old-file vs new-file ambiguity | Common | Mandatory `side` field in format |
| Line drift after edits | Common | Re-validate after any modification |
| Counting errors in long files | Very common | Cap context at ~60 lines per snippet |

**Prompt design for accurate range specification:**

```
Given this diff with line numbers prepended to each line:

```diff
[1] @@ -42,7 +42,12 @@ func validateInput(
[2]  func validateInput(ctx context.Context, input string) error {
[3] -    if input == "" {
[4] -        return errors.New("empty input")
[5] +    if err := checkEmpty(input); err != nil {
[6] +        return fmt.Errorf("validation failed: %w", err)
[7]      }
```

For each segment, provide:
- start_line and end_line (1-based, inclusive, referring to the line numbers shown in brackets)
- side: "old" for deleted lines (prefixed with -), "new" for added lines (prefixed with +)
- anchor_text: first 50 characters of the first line in your range (excluding the line number and +/- prefix)

Verify your line numbers by checking that anchor_text matches the actual content.
```

**Validation feedback format for self-correction:**

Research shows LLMs improve significantly with specific, actionable feedback but effectiveness plateaus after ~5 iterations. Format validation errors as:

```
VALIDATION ERROR at segment "extract-validation":
- DECLARED: pkg/validator.go lines 42-67
- ANCHOR_TEXT: "func validateInput(ctx context.Context"
- ACTUAL content at line 42: "type Config struct {"
- LIKELY ISSUE: Lines shifted; the function starts at line 58

SUGGESTED FIX: Update to start_line: 58, end_line: 83
```

This format provides: (1) what was declared, (2) what was found, (3) diagnosis of the error type, and (4) concrete correction. Studies on LLM self-correction found that "asking LLM to double-check its work" combined with external validation tools (not just self-reflection) produces the best results.

## Recommended implementation architecture

The complete system combines these components:

1. **Diff parser** extracting changed line numbers per file, handling renames/deletes/adds
2. **LLM prompt** with line-numbered diff snippets, format specification, and examples
3. **JSON schema validation** using constrained decoding (Outlines, XGrammar) for format compliance
4. **Anchor text verification** comparing declared vs actual content
5. **Coverage validation** via interval tree algorithm detecting gaps/overlaps
6. **Feedback loop** with max 3-5 iterations, providing specific corrections
7. **Group assignment** post-validation, clustering segments by def-use relationships

The format, validation, and feedback loop work together: the format captures enough redundancy to detect errors, validation identifies specific issues, and feedback helps the LLM converge on correct segments within the research-supported iteration limit.
