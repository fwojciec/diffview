# Two Types of YAGNI

## Feature YAGNI (Keep This)

Don't build capabilities, features, or optimizations you don't need yet. No speculative functionality, no premature scaling, no "we might need this someday" code paths.

**Why:** Features that don't exist can't break, don't need maintenance, and don't constrain future decisions.

**Examples:**
- Don't add caching until you have measured performance problems
- Don't build an admin dashboard until someone actually needs it
- Don't optimize for millions of users when you have hundreds

## Structural YAGNI (Soften This)

The instinct to skip architectural discipline because "it's simple enough." Avoiding dependency injection, cramming responsibilities into one module, hardcoding dependencies, skipping interface boundaries.

**Why it's harmful in the AI era:** When code generation is cheap, the *structure* is what compounds. Each quick regeneration that ignores structural discipline introduces drift. Over many iterations, you get entropy - a codebase that resists modification.

**The paradox:**
- Rigid architectural dogma -> flexible, changeable system
- Loose architectural discipline -> rigid, brittle system

**The Feathers insight:** The cost of change is determined by structural qualities (seams, boundaries, dependency direction), not by feature complexity.

**Maintain even when "you don't need it yet":**
- Dependency injection (even for one implementation)
- Single responsibility per module
- Seams for testing
- Clear interface boundaries
- Consistent patterns across similar components

## When Evaluating Feedback

Ask different questions:
- "Does this add a feature we don't need?" -> Reject (feature YAGNI)
- "Does this add structure that preserves changeability?" -> Accept (reject structural YAGNI)

The beads-review checks whether structural discipline is maintained to make implementing future beads cheap - NOT whether features for future beads were built.
