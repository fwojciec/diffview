# Narrative-Driven Code Change Comprehension: A Research Synthesis

Using LLMs to dynamically shape how code changes are presented—not to judge quality, but to maximize human comprehension—represents a largely unexplored design space. This synthesis draws on cognitive science foundations, practitioner wisdom from elite review cultures, semantic diff tooling, and cross-domain inspiration to identify concrete techniques for building narrative-aware diff viewers.

The core opportunity: **reviewers don't read diffs linearly; they construct mental models opportunistically**. Current tools ignore this, presenting changes as undifferentiated file-ordered hunks. A narrative-driven approach would sequence, chunk, and contextualize changes based on comprehension principles rather than filesystem layout.

---

## Cognitive science reveals how reviewers actually think

**Letovsky's three-layer model** (1987) remains foundational: developers build mental representations at specification ("what"), implementation ("how"), and annotation ("why this maps to that") levels. When reviewing changes, developers generate **why-conjectures** (purpose of a change), **how-conjectures** (mechanism), and **what-conjectures** (classification). A narrative diff viewer should explicitly surface all three layers.

Recent empirical work from Wurzel Gonçalves et al. (2025) observed 10 experienced reviewers conducting 25 real-world code reviews, finding that reviewers build **three distinct mental models simultaneously**: the software system's overall architecture, the specific PR under review, and an "expected model" of what *should* have changed given the context. Mismatches between expected and actual changes drive review attention—suggesting diff tools should highlight surprising changes prominently.

**Miller's 7±2 chunks limit** applies directly to diff review. Pennington's research (1987) showed developers build understanding through chunking: grouping low-level structures into labeled abstractions. As patterns are recognized, **labels replace detail**—"Extract Function refactoring" communicates more than 40 lines of moved code. Sweller's cognitive load theory distinguishes extraneous load (from poor interface design) from intrinsic load (inherent complexity)—diff tools can only reduce the former.

**File order powerfully affects attention**. Fregnan et al. (ESEC/FSE 2022) demonstrated that files appearing later in change sets receive systematically less review effort, with a **linear decline in comment density** as file position increases. Yet reviewers mostly navigate linearly, following the order tools present. This suggests intelligent ordering—by importance, by dependency, by narrative role—could dramatically improve comprehension.

**Beacons** guide expert comprehension. Wiedenbeck's research showed experienced programmers recall "beacon lines"—code that signals familiar patterns (like a swap inside a loop signaling sorting)—far better than non-beacon lines. Expert reviewers use **shallow pattern-matching** when code follows conventions, switching to deep analysis only when expectations are violated. Diff tools could surface beacons explicitly: "This is a standard pagination pattern" or "This implements the Repository interface."

---

## Elite code review cultures encode comprehension principles

**Google's 9-million-review study** (Sadowski et al., ICSE-SEIP 2018) quantified modern code review at scale: **90% of changes touch fewer than 10 files**, median turnaround is **under 4 hours**, and **75%+ of reviews have just one reviewer**. The study revealed that Google optimizes for speed over rigor—trading multiple reviewer perspectives for faster iteration. Their Critique tool integrates static analysis with explicit feedback loops; analyzers generating too many "not useful" clicks get fixed or disabled.

**Linux kernel practices** encode narrative thinking directly. Patches must be submitted in series with a **cover letter (patch 0/n)** explaining the narrative arc. Each patch must compile independently (bisectable), and bug fixes must precede features for backportability. Linus Torvalds emphasizes writing commit messages for outsiders: "Describe what AND why, not how—the diff shows how." The kernel's structured tags (`Fixes:`, `Reported-by:`, `Tested-by:`) create machine-readable narrative metadata.

The **50/72 rule** (50-character subjects, 72-character body lines) originated from Tim Pope's 2008 analysis of kernel commits but has become universal. More important is the **imperative mood convention** ("Fix bug" not "Fixed bug")—framing commits as commands to be applied rather than diary entries about what happened.

**Conventional Commits** specification (conventionalcommits.org) maps commit types to semantic versioning: `fix:` → PATCH, `feat:` → MINOR, `BREAKING CHANGE:` → MAJOR. This creates **machine-parseable narrative structure**: changelogs, version bumps, and release notes can be auto-generated. The spec includes types for docs, style, refactor, test, and chore—a coarse but practical taxonomy.

**Microsoft's research** yielded a provocative finding: "Code reviews do not find bugs" (ICSE-SEIP 2015). The primary value lies elsewhere—knowledge transfer, design discussion, architectural coherence. This reframes what a comprehension-focused tool should optimize for: not bug-finding efficiency, but understanding transfer.

---

## Semantic diff tools solve presentation problems that matter

**The line-based/structural/semantic spectrum** represents increasing levels of language awareness:

| Approach | Unit of Comparison | Whitespace Handling | Move Detection | Refactoring Awareness |
|----------|-------------------|---------------------|----------------|----------------------|
| **Line-based (diff)** | Text lines | Shows all changes | None | None |
| **Structural (Difftastic)** | AST nodes | Filters syntax-irrelevant | Yes | Limited |
| **Semantic (SemanticDiff)** | AST + semantic rules | Filters + language invariances | Yes + grouping | Explicit (renames, moves) |

**Difftastic** (open source, tree-sitter based) treats structural diffing as a graph problem using Dijkstra's algorithm. It understands when whitespace matters semantically (Python indentation) versus cosmetically. The explicit non-goal of producing machine-applicable patches is instructive: the output optimizes for **human reading, not machine processing**.

**SemanticDiff** goes further with language-specific invariance rules. Python keyword argument reordering (`foo(a=1, b=2)` ≡ `foo(b=2, a=1)`) is semantically identical; SemanticDiff hides such changes. Their philosophy: "Let the CI validate code style, while you concentrate on logic changes."

**GumTree** (Falleri et al., ASE 2014) established the academic foundation with its two-phase algorithm: greedy top-down matching of isomorphic subtrees, then bottom-up recovery of remaining mappings. Critically, the paper notes the goal is not algorithmically optimal edit scripts but scripts "reflecting developer intent." GumTree 2.0 (ICSE 2024) introduced heuristics yielding **50% smaller edit scripts** while scaling to large ASTs.

**RefactoringMiner** (Tsantalis et al., TSE 2020) detects **40 refactoring types** at 99.6% precision without requiring code similarity thresholds. This enables separating "what was refactored" from "what behavior changed"—Fowler's "two hats" made concrete.

**ClDiff** (ASE 2018) groups fine-grained AST diffs at statement level with five pre-defined link types. A human study with 10 participants confirmed usefulness for code review. This represents rare empirical validation of presentation choices.

---

## Adjacent fields offer patterns worth borrowing

**Legal redlining** produces a third document showing all changes—a "Changes Report" listing modifications in tabular form separate from inline markup. The concept of **cumulative diff** (comparing first and final versions, collapsing intermediate noise) directly applies to multi-round code reviews. Lawyers view change detection as **risk mitigation**, framing that could inform how diff tools highlight significant changes.

**Academic peer review** requires dual artifacts: clean version plus tracked changes. More importantly, the **Response to Reviewers** document provides structured point-by-point justification: each numbered reviewer comment receives a numbered response explaining what was changed and why (or why not). This maps directly to code review comment resolution.

**Figma's version history** collapses autosaves between named milestones, reducing noise while preserving granular history. Named versions get **titles (25 chars) + descriptions (140 chars)** for context. Non-destructive restore lets designers safely explore history. These patterns apply directly to commit squashing and force-push scenarios.

**Google Docs' mode indicator** (Editing/Suggesting/Viewing) provides clear state awareness. The **Commenter role** forces Suggestion mode—a permission-enforced interaction pattern ensuring feedback without direct modification. View mode toggling (Simple Markup/All Markup/No Markup) manages cognitive load during different review phases.

---

## Change classification taxonomies provide vocabulary

**Swanson's classic taxonomy** (1976) remains influential: **Corrective** (bug fixes), **Adaptive** (environment changes), **Perfective** (enhancements). ISO/IEC 14764 added **Preventive** (maintainability improvements). Research found perfective changes dominate at **60%** of maintenance effort.

**Fowler's refactoring catalog** (68+ named patterns) provides shared vocabulary that communicates intent: "Extract Function" conveys more than describing which lines moved. The catalog's organization—Basic, Encapsulation, Moving Features, Organizing Data, Simplifying Conditionals, Refactoring APIs, Inheritance—maps to comprehension strategies.

**Narrative arc patterns** for changes:
- **Cause → Effect** (Bugfix): Problem → Investigation → Root cause → Fix → Verification
- **Core → Periphery** (Feature): Central implementation → Supporting changes → Integration
- **Before → After** (Refactoring): Old structure → Transformation → New structure
- **Debt Payment**: Historical compromise → Growing pain → Proper solution

---

## AI considerations point toward augmenting comprehension, not automating judgment

**Automation bias in code review** is documented and dangerous. Microsoft Research found that adding explanations to AI decisions "does not appear to reduce overreliance and some studies suggest it might even increase it." Reviewers develop heuristics about when to trust AI rather than engaging analytically with each recommendation.

A study on automated code review found reviewers saying: "It may create bias so reviewers may ignore by saying that if any other issue exists, the bot would have written it." This **diffusion of responsibility** allows severe bugs to go unnoticed.

**Cognitive forcing functions** (Buçinca et al., CHI 2021) can reduce overreliance: requiring prediction before seeing AI output, delaying suggestions, requiring explanation review. The trade-off: interventions that reduce overreliance most receive **least favorable user ratings**.

The distinction for a narrative-driven diff viewer: use AI to **surface context, not make judgments**. Appropriate uses include:
- Summarizing changes (as drafts for human editing)
- Explaining unfamiliar code patterns
- Surfacing related changes and dependencies
- Generating documentation stubs
- Detecting and labeling refactoring patterns

**Comprehension debt**—the future cost of understanding AI-generated code not fully comprehended at creation—accelerates with AI-assisted development. CodeRabbit (2025) found AI-generated PRs contain **1.7x more issues** than human-written code. The "almost right" phenomenon (66% of developers describe AI code this way) creates particular review challenges.

---

## Annotated bibliography

### Cognitive Science Foundations

**Letovsky, S. (1987). "Cognitive processes in program comprehension." Journal of Systems and Software, 7(4), 325-339.**
Foundational model of specification/implementation/annotation layers. Identified why/how/what conjectures as drivers of comprehension. Essential reading.

**Pennington, N. (1987). "Stimulus structures and mental representations in expert comprehension of computer programs." Cognitive Psychology, 19(3), 295-341.**
Demonstrated developers build Program Model (control-flow) before Situation Model (functional). Established chunking as core comprehension mechanism.

**Von Mayrhauser, A. & Vans, A.M. (1995). "Program comprehension during software maintenance and evolution." IEEE Computer, 28(8), 44-55.**
Integrated metamodel showing programmers switch between top-down, bottom-up, and situation models dynamically. Key insight: comprehension is opportunistic.

**Soloway, E. & Ehrlich, K. (1984). "Empirical Studies of Programming Knowledge." IEEE TSE, SE-10(5), 595-609.**
Defined "programming plans" and "rules of discourse." Demonstrated experts perform far better on conventional code than unconventional code with identical logic.

**Wiedenbeck, S. (1986). "Beacons in computer program comprehension." IJMMS, 25(6), 697-709.**
Identified beacon lines as comprehension anchors. Experts recalled beacon lines far better than non-beacon lines. Implications for highlighting.

**Wurzel Gonçalves, P. et al. (2025). "Code Review Comprehension: Reviewing Strategies Seen Through Code Comprehension Theories." arXiv:2503.21455.**
Recent empirical study observing 10 reviewers on 25 real PRs. Extended Letovsky for code review. Identified chunking strategies and tool design recommendations.

**Fregnan, E. et al. (2022). "First come first served: the impact of file position on code review." ESEC/FSE 2022, 483-494.**
Demonstrated linear decline in review effort for later-positioned files. Critical implication: intelligent file ordering matters.

### Code Review Research

**Sadowski, C. et al. (2018). "Modern Code Review: A Case Study at Google." ICSE-SEIP 2018.**
Analysis of 9 million reviews. Median turnaround <4 hours, 90% of changes <10 files, 75%+ single reviewer. Benchmark for modern code review.

**Bacchelli, A. & Bird, C. (2013). "Expectations, Outcomes, and Challenges of Modern Code Review." ICSE 2013.**
"Code review is understanding"—the second most frequent activity is comprehension (clarification questions, rationale doubts).

**Czerwonka, J., Greiler, M., & Tilford, J. (2015). "Code Reviews Do Not Find Bugs." ICSE-SEIP 2015.**
Provocative Microsoft finding: reviews' value is knowledge transfer and design discussion, not defect detection.

**Bosu, A. et al. (2015). "Characteristics of Useful Code Reviews." MSR 2015.**
Analysis of 1.5 million Microsoft review comments. Most comments concern structure and style, not bugs. Useful comments identify functional issues.

### Semantic Differencing

**Falleri, J.-R. et al. (2014). "Fine-grained and accurate source code differencing." ASE 2014, 313-324.**
GumTree paper. Two-phase matching algorithm. Goal: edit scripts reflecting developer intent, not algorithmic optimality. Highly cited foundation.

**Falleri, J.-R. & Martinez, M. (2024). "Fine-grained, accurate and scalable source differencing." ICSE 2024.**
GumTree 2.0 with 50% smaller edit scripts and better scaling.

**Fluri, B. et al. (2007). "Change Distilling: Tree Differencing for Fine-Grained Source Code Change Extraction." IEEE TSE, 33(11), 725-743.**
Defined 35 fine-grained change types. Improved edit script approximation by 45%.

**Tsantalis, N. et al. (2020). "RefactoringMiner 2.0." IEEE TSE 2020.**
Detects 40 refactoring types at 99.6% precision without similarity thresholds. Enables separating refactoring from behavior change.

**Huang, K. et al. (2018). "ClDiff: Generating Concise Linked Code Differences." ASE 2018, 679-690.**
Groups fine-grained diffs at statement level with linking. Rare example of human study validating presentation choices.

### Practitioner Resources

**Tim Pope (2008). "A Note About Git Commit Messages." tbaggery.com.**
Origin of 50/72 rule. Influential formatting conventions.

**Chris Beams. "How to Write a Git Commit Message."**
Seven rules synthesis. Widely referenced practitioner guide.

**Conventional Commits Specification. conventionalcommits.org.**
Machine-parseable commit taxonomy mapping to semantic versioning.

**Linux kernel documentation: "Submitting patches." kernel.org.**
Patch series structure, cover letters, structured tags. Primary source for narrative patch culture.

**Google SWE Book, Chapter 19: "Critique: Google's Code Review Tool." abseil.io.**
Design philosophy behind Google's internal review tooling.

### AI and Comprehension

**Buçinca, Z. et al. (2021). "To Trust or to Think: Cognitive Forcing Functions Can Reduce Overreliance on AI Systems During AI-assisted Decision Making." CHI 2021.**
Interventions reducing overreliance work but reduce user satisfaction. Key design tension.

**Microsoft Research (2022). "Overreliance on AI: Literature Review."**
Comprehensive synthesis of automation bias research. Explanations don't reduce overreliance.

**CodeRabbit (2025). "State of AI vs Human Code Generation."**
AI-generated PRs contain 1.7x more issues than human code. Quantifies comprehension debt risk.

---

## Taxonomy of approaches

### Presentation Strategies

| Strategy | Advantages | Disadvantages | Best For |
|----------|-----------|---------------|----------|
| **File-ordered** | Familiar, matches filesystem | Ignores semantic relationships | Simple changes, single-file |
| **Dependency-ordered** | Shows cause before effect | Complex to compute | Multi-file features |
| **Importance-ordered** | Core changes get attention | "Importance" is subjective | Large changesets |
| **Commit-by-commit** | Shows evolution narrative | Requires good git hygiene | Feature development |
| **Semantic grouping** | Groups related operations | May fragment files confusingly | Refactorings |

### Diff Granularity

| Level | Shows | Hides | Tools |
|-------|-------|-------|-------|
| **Line-based** | All text changes | Nothing | git diff, GitHub |
| **Word-level** | Word changes within lines | Nothing | git diff --word-diff |
| **AST/Structural** | Syntax element changes | Whitespace, formatting | Difftastic |
| **Semantic** | Meaning changes | Syntax-equivalent variations | SemanticDiff |
| **Refactoring-aware** | Behavior changes + labeled refactorings | Behavior-preserving transformations | RefactoringMiner |

### Context Disclosure

| Approach | Shows Initially | Reveals On Demand | Trade-off |
|----------|----------------|-------------------|-----------|
| **Full context** | All surrounding code | N/A | Complete but overwhelming |
| **Unified diff (3-line)** | 3 lines context | N/A | Standard but minimal |
| **Progressive disclosure** | Hunks only | Surrounding code, file, history | Cleaner but requires interaction |
| **Narrative summary first** | Description + key changes | Full diff | Fast orientation, two-pass review |

---

## Gaps and opportunities

### Gap 1: No tool presents changes by narrative role
Current tools order by filename or commit sequence. No tool sequences hunks by comprehension logic: "Show me the core behavior change first, then the supporting infrastructure, then the tests."

**Opportunity**: LLM-powered "story ordering" that resequences hunks based on detected narrative role (setup, core change, consequences, verification).

### Gap 2: Cross-file semantic operations are invisible
Moving a function between files appears as deletion + insertion. Extracting a class into a new file shows as unrelated changes. RefactoringMiner detects these but isn't integrated into review tools.

**Opportunity**: First-class "operation view" showing "Extract Class from UserService to UserValidator" as a single logical operation spanning files.

### Gap 3: Expected vs actual model mismatch isn't surfaced
Wurzel Gonçalves showed reviewers build "expected models" of what should change. Surprising changes drive attention. No tool highlights: "Based on the PR description, you might expect X, but these files were also modified."

**Opportunity**: LLM-powered expectation checking that flags changes inconsistent with stated intent or surprising given typical patterns.

### Gap 4: No tool separates refactoring from behavior change
Fowler's "two hats" principle (don't mix refactoring with behavior change) is universally known but unsupported by tooling. Mixed commits require reviewers to mentally untangle.

**Opportunity**: Automatic "layer separation" showing refactoring changes separately from behavior changes, with confidence levels.

### Gap 5: Chunk boundaries don't respect cognition
Files are arbitrary; functions are arbitrary; even commits may not represent natural comprehension units. Miller's 7±2 applies but nothing enforces it.

**Opportunity**: Dynamic chunking that groups changes into cognitively-manageable units based on detected relationships, not filesystem structure.

### Gap 6: Limited empirical validation of presentation choices
ClDiff's 10-participant study is exceptional. Most tools launch without human studies. The relationship between presentation decisions and comprehension outcomes is largely unmeasured.

**Opportunity**: Instrument a narrative diff viewer to measure comprehension time, error rates, and subjective confidence across presentation strategies.

### Gap 7: No "Response to Reviewers" equivalent for code
Academic publishing's structured point-by-point response document doesn't exist in code review. Comments get resolved with ambiguous "Done" markers.

**Opportunity**: Structured revision tracking linking each review comment to specific code changes with explicit justification.

---

## Ideas to steal and adapt

### From Legal Redlining
1. **Changes Report**: Generate a separate tabular summary of all modifications before showing inline diff
2. **Cumulative diff mode**: Compare initial submission to current version, collapsing intermediate revisions
3. **Risk framing**: Highlight "areas of concern" based on change characteristics (size, complexity, security-sensitive patterns)

### From Academic Publishing
4. **Numbered point-by-point responses**: Each review comment gets a number; each revision response references it with "Addressed by [link to hunk]" or "Not addressed because [reason]"
5. **Clean + marked dual view**: Provide both "final result" and "what changed" views as distinct artifacts

### From Figma
6. **Milestone collapsing**: Auto-collapse commits between tagged versions; expand on demand
7. **Version notes**: Encourage 25-char title + 140-char description for each meaningful commit
8. **Non-destructive exploration**: Show history as a timeline; clicking any point shows that version without affecting current state

### From Collaborative Writing
9. **Mode indicator**: Clear visual signal of current view state (all changes / significant only / clean)
10. **Batch operations**: Accept/reject groups of related changes together, not individually

### From Cognitive Science
11. **Beacon highlighting**: Detect and label common patterns ("This is a pagination pattern", "Standard error handling")
12. **Chunking by operation**: Present changes in 5-7 logical groups, not arbitrary files
13. **Core-first ordering**: Detect the "main" change and show it first; supporting changes follow

### From Semantic Diff Tools
14. **Refactoring detection**: Integrate RefactoringMiner-style detection; show "Extract Function" not raw line changes
15. **Semantic filtering toggle**: Switch between "all changes" and "behavior changes only"
16. **AST-level explanation**: When presenting AST diffs, provide natural-language summary ("This adds a null check before the method call")

### From Linux Kernel Culture
17. **Cover letter view**: For multi-commit PRs, present a synthesized "cover letter" explaining the patch series narrative
18. **Bisectability indicator**: Show whether each commit compiles independently
19. **Structured tags display**: Parse and prominently display Fixes:, Related:, Depends-on: relationships

### From AI Research
20. **Cognitive forcing**: Before showing AI-generated summaries, briefly show the raw diff to build independent mental model
21. **Uncertainty visualization**: When AI summarizes changes, indicate confidence levels; highlight areas of uncertainty
22. **Context surfacing**: Use AI to find and present related code, similar past changes, relevant documentation—not to judge quality

---

## People and communities to follow

### Researchers

**Alberto Bacchelli** (University of Zurich) — Co-author of Google code review study; leads research on code review comprehension and tooling. Prolific producer of empirical code review research.

**Michaela Greiler** — Former Microsoft Research on code review; now runs awesomecodereviews.com. Bridges academic and practitioner perspectives.

**Jean-Rémy Falleri** — GumTree creator; active researcher on program differencing at University of Bordeaux.

**Nikolaos Tsantalis** — RefactoringMiner creator at Concordia University. Leading researcher on refactoring detection.

**Martin Monperrus** — Maintains curated list of tree differencing resources (monperrus.net/martin/tree-differencing). KTH researcher on program repair and differencing.

**Margaret-Anne Storey** (University of Victoria) — Decades of research on program comprehension and developer tools.

**Pavlína Wurzel Gonçalves** — Lead author of recent Code Review Comprehension study extending Letovsky's model.

### Practitioners and Projects

**Difftastic** (Wilfred Hughes) — Best-documented open-source structural diff. Design decisions are well-explained in docs and blog posts.

**SemanticDiff** — Commercial semantic diff with detailed documentation explaining design philosophy.

**Conventional Commits community** — Active spec development for structured commit messages.

**Git mailing list and Linux kernel-mentors** — Primary source for understanding patch series culture.

**GitHub's diff rendering team** — Occasional blog posts on diff presentation decisions.

### Communities

**ICSE (International Conference on Software Engineering)** — Premier venue for code review and comprehension research. SEIP (Software Engineering in Practice) track especially relevant.

**FSE/ESEC (Foundations/European Software Engineering Conference)** — Strong empirical software engineering research including code review.

**ASE (Automated Software Engineering)** — Venue for tool-building research including semantic differencing.

**MSR (Mining Software Repositories)** — Empirical studies using commit/review data.

**Strange Loop** — Practitioner talks sometimes cover developer tooling and comprehension.

**r/programming and Hacker News** — Practitioner discussion of diff tools and code review practices.

---

## Synthesis: design principles for narrative-driven diff viewers

Based on this research synthesis, a narrative-driven diff viewer should:

1. **Lead with intent, not files** — Show what the change accomplishes before showing how files changed. Generate or extract a "cover letter" summarizing the narrative arc.

2. **Order by comprehension, not alphabet** — Sequence hunks by narrative role (core change → supporting infrastructure → tests → cleanup) rather than filename.

3. **Separate layers explicitly** — Use refactoring detection to present behavior-preserving transformations separately from behavior changes. Enable toggling between layers.

4. **Respect cognitive limits** — Chunk changes into 5-7 logical groups. Provide progressive disclosure: summary → groups → files → hunks → lines.

5. **Surface the unexpected** — Compare stated intent against actual changes. Highlight modifications inconsistent with PR description or surprising given patterns.

6. **Provide vocabulary** — Label detected patterns: "This appears to be an Extract Function refactoring" or "Standard pagination pattern detected."

7. **Support non-linear exploration** — Let reviewers jump to what interests them, but provide recommended reading order. Track what's been reviewed.

8. **Link comments to changes bidirectionally** — Enable structured "Response to Reviewers" tracking: each comment maps to specific code changes with explicit justification.

9. **Use AI for presentation, not judgment** — Generate summaries (as drafts), surface context, detect patterns—but frame all AI output as suggestions requiring verification, not authoritative assessments.

10. **Measure and iterate** — Instrument comprehension time, error detection rates, and reviewer confidence. Use data to validate presentation decisions.

The distinctive opportunity: where semantic diff tools solve the problem of *what* changed at a technical level, narrative-driven tools solve *how to present* changes for human comprehension. This is fundamentally a cognitive design problem, not an algorithmic one—and LLMs offer new capabilities for dynamically shaping presentation based on detected change characteristics.
