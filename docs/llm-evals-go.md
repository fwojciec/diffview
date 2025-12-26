# Go-centric LLM evaluation: A practical infrastructure guide

**The Go ecosystem lacks mature LLM evaluation frameworks comparable to Python's promptfoo or DeepEval, but a practical path exists.** The most battle-tested approach extends Go's native testing framework with LLM-as-judge assertions—exemplified by Mattermost's production system. For structured output validation, Go actually excels thanks to strong typing and JSON schema generation from structs. Since you're already building with Lipgloss and Chroma, a TUI-based review workflow will be more idiomatic than notebooks, and modernc.org/sqlite combined with JSONL files provides the ideal data layer for iterative improvement.

## Go lacks dedicated eval frameworks but has workable alternatives

No Go-native equivalent to promptfoo, ragas, or deepeval exists with comparable feature sets. However, two emerging options merit attention:

**maragu.dev/gai/eval** provides basic evaluation integrated with `go test`. It offers lexical similarity (Levenshtein, exact match) and semantic similarity (cosine distance with embeddings) scorers, logging results to `evals.jsonl` for tracking over time. The LLM-as-judge scorer is planned but not yet available. The philosophy mirrors TDD: "Evaluation-Driven Development."

**The Mattermost pattern** represents the most production-proven approach. They extended Go's testing framework to support LLM evaluation with rubric-based assertions:

```go
func TestStoryGeneration(t *testing.T) {
    evals.Run(t, "summarize_diff", func(e *evals.EvalT) {
        result := generateStory(gitDiff)
        require.NotEmpty(t, result)
        evals.LLMRubricT(e, "identifies the main code change", result)
        evals.LLMRubricT(e, "uses appropriate technical terminology", result)
    })
}
```

This integrates naturally with CI/CD via `GOEVALS=1 go test`, making evals opt-in during normal development but enforced in pipelines. Mattermost also built a Bubble Tea viewer for browsing results—directly applicable to your Lipgloss-based stack.

For structured output validation, Go's type system becomes an advantage. The **sashabaranov/go-openai** library includes `jsonschema.GenerateSchemaForType()` which generates JSON schemas from Go structs for OpenAI's structured outputs mode. Combined with `VerifySchemaAndUnmarshal()`, you get type-safe validation in one step. For more complex validation, **santhosh-tekuri/jsonschema/v5** provides full JSON Schema draft 2020-12 compliance, while **godantic** handles streaming JSON with partial validation—useful if your LLM responses stream.

## A TUI review workflow fits Go idioms better than notebooks

GoNB has matured significantly (v0.11.3, December 2025, 968 GitHub stars) with compiled Go cells, gopls integration, and Jupyter compatibility. However, for your use case—reviewing LLM-generated stories about code changes—**a TUI built with your existing Charm stack is more idiomatic and practical.**

You're already using Lipgloss for styling. The natural extension is:

- **Bubble Tea** (now at 1.0 stable) for the application framework using Elm Architecture
- **Bubbles** for table views, viewports, and pagination of eval results  
- **Huh** for annotation forms—capturing labels, confidence scores, and notes
- **Chroma** (which you have) for syntax-highlighted diff display

A practical architecture for reviewing generated stories:

```
┌─────────────────────────────────────────────┐
│ Diff Viewer (Chroma syntax highlighting)   │
├─────────────────────────────────────────────┤
│ Generated Story (scrollable viewport)       │
├─────────────────────────────────────────────┤
│ Annotation Form (Huh)                       │
│  - Quality: [1-5 select]                    │
│  - Accuracy: [checkbox]                     │
│  - Notes: [text input]                      │
├─────────────────────────────────────────────┤
│ [←Prev] [→Next] [Save] [Skip] [q]uit       │
└─────────────────────────────────────────────┘
```

If you need quick tabular data exploration without building UI, **tview** offers ready-made Table and TreeView widgets. For occasional notebook-style exploration, GoNB works with standard Jupyter and Google Colab—install via `go install github.com/janpfeifer/gonb@latest && gonb --install`.

## SQLite plus JSONL provides the ideal data layer

Go projects follow established patterns for eval data that balance queryability with human readability.

**For eval records requiring queries:** Use SQLite with **modernc.org/sqlite** (pure Go, no CGO, cross-compilation friendly). It's roughly 2x slower than mattn/go-sqlite3 for inserts but avoids C compiler requirements. Store structured data in JSON columns—SQLite 3.45+ supports JSONB for efficient querying:

```go
db.Exec(`CREATE TABLE evals (
    id INTEGER PRIMARY KEY,
    run_id TEXT,
    input_hash TEXT,
    output TEXT,
    scores TEXT,  -- JSON: {"accuracy": 0.85, "completeness": 0.92}
    human_review TEXT,  -- JSON or NULL
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)`)
```

**For append-only logs and version control:** Use JSONL files with **olivere/ndjson** or the standard library's `json.NewDecoder`. JSONL files are git-friendly, human-readable, and trivial to process:

```go
// Append eval result
f, _ := os.OpenFile("evals.jsonl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
json.NewEncoder(f).Encode(result)
```

**Golden file testing** uses the established `testdata/*.golden` pattern. **sebdah/goldie** is the most popular library, supporting colored diffs and auto-update via `go test -update`:

```go
g := goldie.New(t, goldie.WithFixtureDir("testdata"))
g.Assert(t, "expected_story", actualOutput)
```

For your code analysis tool, a practical hybrid: SQLite for queryable eval state (which examples need review, aggregate scores across runs), JSONL for immutable run logs, and golden files for regression tests of expected story outputs.

## Ollama and LocalAI offer limited eval patterns to borrow

**Ollama's approach** centers on integration tests in `integration/` using Go's build tag system (`go test -tags=integration`). Tests spawn a server, run requests against multiple model architectures, and validate responses. However, their testing focuses on infrastructure correctness rather than output quality evaluation. Notable: they acknowledge flaky tests ("sometimes generates no response on first query") and use environment variables for test configuration.

External quality tools have emerged around Ollama: **ollama-benchmark** uses LLM-as-Judge with MT-Bench datasets, **ollama-grid-search** (Rust/React) enables A/B testing prompts across models. These confirm the pattern: infrastructure teams offload quality evaluation to separate tooling.

**LocalAI** relies on CI-driven testing across model backends (rwkv, cerebras, whisper, bert embeddings) but similarly lacks sophisticated output quality metrics.

The takeaway: even major Go LLM projects use basic integration testing for infrastructure and rely on external tools or custom solutions for quality evaluation. There's no comprehensive solution to adopt wholesale.

## The minimal viable Go-centric eval stack

For your code analysis tool generating stories from git diffs, here's the **80/20 implementation** requiring roughly 500-800 lines of Go:

**Core components:**

1. **Test case structure** matching your domain:
```go
type EvalCase struct {
    DiffPath     string   `json:"diff_path"`     // Path to git diff file
    Expected     string   `json:"expected"`      // Golden story (optional)
    Rubrics      []string `json:"rubrics"`       // LLM-as-judge criteria
    Tags         []string `json:"tags"`          // Filter by change type
}
```

2. **LLM-as-judge runner** using your existing OpenAI/Anthropic client:
```go
func JudgeWithRubric(llm Client, rubric, output string) (passed bool, reasoning string) {
    prompt := fmt.Sprintf(`Evaluate if this output satisfies the criterion.
Criterion: %s
Output: %s
Respond with JSON: {"passed": bool, "reasoning": "..."}`, rubric, output)
    // Parse structured response
}
```

3. **Result storage** as described above (SQLite + JSONL)

4. **TUI reviewer** extending your existing Lipgloss/Chroma stack

**What to delegate to Python via subprocess:**

- Semantic similarity scoring (sentence-transformers embeddings)
- Complex metrics if needed later (ROUGE, BLEU for text comparison)
- Synthetic test case generation

The subprocess pattern is trivial:
```go
cmd := exec.Command("python", "score_similarity.py")
cmd.Stdin = strings.NewReader(jsonInput)
output, _ := cmd.Output()
```

## Trade-offs favor Go for orchestration, Python for specialized metrics

| Dimension | Go-native | Python subprocess | Full Python stack |
|-----------|-----------|-------------------|-------------------|
| **CI/CD integration** | Native `go test` | Extra step | Separate workflow |
| **Type safety** | Strong | At boundary | None in Go |
| **Semantic metrics** | Limited | Full access | Full access |
| **Team context switching** | None | Minimal | Significant |
| **Deployment** | Single binary | Binary + Python | Python environment |

Assembled (millions of monthly LLM requests in Go) articulates the practical pattern: *"We often prototype features entirely in Python, then gradually port performance-critical components to Go once they're proven."* For eval infrastructure specifically, they maintain a lightweight Python service for ML-specific tasks while keeping core infrastructure in Go.

**Concrete recommendation for your situation:** Build the eval runner, assertion framework, and TUI reviewer in Go. Use Python subprocess calls only when you need embedding-based similarity (which may not be necessary if LLM-as-judge with rubrics suffices for your story quality assessment). The Mattermost pattern provides a template that integrates naturally with Go's testing ecosystem and your existing toolchain.

The ecosystem gap is real—Go lacks 60+ eval metrics, red teaming tools, and visualization dashboards that Python frameworks provide. But for your focused use case (iteratively improving LLM outputs that analyze git diffs), a minimal custom stack will be more maintainable than fighting against Python tooling while staying in Go for your TUI application.
