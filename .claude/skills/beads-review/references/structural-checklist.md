# Structural Review Checklist

Evaluate code changes against these criteria. Focus on structural qualities that affect future changeability, not correctness (that's handled by the standard code review).

## Dependency Injection

**Pass**: Dependencies passed as parameters or constructor arguments
**Fail**: Hardcoded instantiation of external dependencies, global state access

```go
// GOOD: Injected
func NewService(db Database, logger Logger) *Service

// BAD: Hardcoded
func NewService() *Service {
    db := sql.Open("postgres", os.Getenv("DB_URL"))
}
```

## Single Responsibility

**Pass**: Each type/function has one reason to change
**Fail**: Types mixing concerns (e.g., business logic + HTTP handling + persistence)

Watch for:
- Functions doing multiple unrelated things
- Types with fields from different domains
- Methods that could be split into separate interfaces

## Testing Seams

**Pass**: Behavior can be tested via interfaces or function parameters
**Fail**: Behavior requires real infrastructure or global state to test

Key indicators:
- Can this be tested with a mock?
- Are side effects isolated to injected dependencies?
- Can edge cases be exercised without complex setup?

## Interface Boundaries

**Pass**: Clear contracts between components, minimal surface area
**Fail**: Leaky abstractions, exposing implementation details

Check:
- Do interfaces expose only what consumers need?
- Are internal types kept internal?
- Would changing implementation require changing callers?

## Codebase Consistency

**Pass**: Follows existing patterns in the codebase
**Fail**: Introduces new patterns without justification

Verify against:
- Existing package structure (Ben Johnson layout)
- Naming conventions
- Error handling patterns
- Test organization

## Forward Compatibility

**Pass**: No assumptions that conflict with known downstream work
**Fail**: Structural choices that will require refactoring for upcoming issues

Questions:
- Does this lock in decisions that downstream work needs flexibility on?
- Are extension points present where downstream work will need them?
- Would a different approach make downstream work trivially easier?
