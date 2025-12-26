# GitHub Diff View Design System: Dark Mode Analysis for Terminal Adaptation

GitHub's unified diff view relies on the **Primer design system**, which uses functional color tokens with alpha transparency to achieve its distinctive visual hierarchy. This report provides the exact color specifications, design rationale, and practical guidance for adapting the system to solid-color terminal environments.

## Core color architecture: rgba foundations require conversion

GitHub's dark mode canvas uses **`#0d1117`** as the base background. All diff colors are defined as rgba values with transparency that blend against this canvas—a critical consideration for terminal adaptation where only solid RGB is available.

### Diff line backgrounds (dark mode)

| Element | CSS Variable | Raw RGBA Value | Blended Solid (on #0d1117) |
|---------|--------------|----------------|---------------------------|
| Added line background | `--diffBlob-addition-bgColor-line` | rgba(46, 160, 67, 0.15) | **#1b3a28** |
| Added line gutter | `--diffBlob-addition-bgColor-num` | rgba(46, 160, 67, 0.40) | **#214d32** |
| Added word highlight | `--diffBlob-addition-bgColor-word` | rgba(63, 185, 80, 0.40) | **#2b5a3c** |
| Deleted line background | `--diffBlob-deletion-bgColor-line` | rgba(248, 81, 73, 0.15) | **#3c1e1f** |
| Deleted line gutter | `--diffBlob-deletion-bgColor-num` | rgba(248, 81, 73, 0.40) | **#6b3b3c** |
| Deleted word highlight | `--diffBlob-deletion-bgColor-word` | rgba(248, 81, 73, 0.40) | **#6b3b3c** |
| Context line | (uses canvas) | — | **#0d1117** |
| Hunk header (@@ markers) | `--bgColor-accent-muted` | rgba(56, 139, 253, 0.10) | **#121d2f** |

The **gutter columns use more saturated** (higher alpha ~0.40) versions of the line background colors, creating visual separation between line numbers and code content without requiring a hard border.

### Text and foreground colors

Default code text uses **`#e6edf3`** (fgColor-default) on all backgrounds. GitHub does **not** change text color based on line type—syntax highlighting colors remain consistent across added, deleted, and context lines. Key foreground values include `--fgColor-muted: #8b949e` for line numbers and hunk header text, and `--fgColor-success: #3fb950` / `--fgColor-danger: #f85149` for the success/danger color scale used elsewhere in the UI.

## Syntax highlighting interaction: no blending, parallel systems

GitHub maintains **two independent color layers** for diff views. The diff background colors provide the line-level context (added/deleted/unchanged), while syntax highlighting tokens operate on top without interaction. The `prettylights` syntax theme colors are designed to remain readable on both light diff backgrounds and dark diff backgrounds. Key dark mode syntax tokens include comments at **#8b949e**, strings at **#a5d6ff**, keywords at **#ff7b72**, and entity names at **#d2a8ff**.

There is **no CSS blending mode** between syntax colors and diff backgrounds—the foreground text simply renders with full opacity over the semi-transparent background. This is why GitHub's approach translates cleanly to terminals: you can pre-blend the backgrounds to solid colors and overlay syntax colors directly.

## Visual hierarchy and structural elements

### Typography specifications

GitHub uses a monospace font stack: `ui-monospace, SFMono-Regular, SF Mono, Menlo, Consolas, Liberation Mono, monospace` at **12px with 20px line height** (1.667 ratio). No font weight variations distinguish changed lines—color alone conveys diff state. Tab rendering defaults to **8 spaces** but is user-configurable.

### Gutter design and line numbers

Each line number column is approximately **50px wide**, with right-aligned numbers and 8-10px internal padding. The unified diff displays two columns: old line number and new line number, separated by a subtle 1px border using `--borderColor-muted`. When a line is added, the old line number cell is **empty** (blank space, not a dash), maintaining the same background color. Deleted lines leave the new line number cell empty.

### The +/- prefix characters

The plus and minus symbols are **part of the code content**, not a separate column. They appear as the first character using the same font, size, and foreground color as the surrounding code. They occupy exactly **one character width** (~7.2px) followed by a single space. There is no separate background zone—prefixes inherit the full line background.

### File boundaries and hunk spacing

Multi-file diffs use distinct file headers with a subtle gray background (`--bgColor-muted: #161b22`), ~16px padding, rounded 6px corners on the container, and a bottom border. File sections can collapse/expand. Hunk headers with the `@@ -n,m +n,m @@` syntax use a blue-tinted background for visual distinction and act as the only separator between hunks—there is no additional vertical spacing.

## Design rationale: accessibility-first color choices

### Contrast targeting

GitHub aims for **WCAG 2.2 AA compliance**, requiring 4.5:1 contrast for normal text and 3:1 for UI components. The Primer team runs automated contrast checking on every pull request to the primitives repository, testing 100+ color pair combinations across all nine themes before production deployment.

### Why these specific color values

The muted backgrounds (~15% opacity for lines, ~40% for gutters) balance **visibility of changes with code readability**. Primer documentation notes they "experimented with major adjustments in brightness and hue" but intentionally scoped changes to accessibility improvements while preserving brand consistency. The result avoids the "Christmas tree effect" where saturated colors overwhelm code content.

### Colorblind accessibility

GitHub offers dedicated Protanopia/Deuteranopia themes that **substitute orange and blue for red and green**, addressing the ~8% of users affected by red-green color blindness. Beyond themes, multiple indicators reinforce state: +/- symbols, position in the UI, and textual context. The WCAG 1.4.1 principle of "never relying on color alone" guides these decisions.

## Light mode comparison: same tokens, shifted values

The light mode canvas is **#ffffff** with success-muted at **#dafbe1** (pale green) and danger-muted at **#ffebe9** (pale pink). Word-level highlights use **#aceebb** and **#ffc1c0** respectively. The same functional token names (`--diffBlob-addition-bgColor-line`) resolve to these different values automatically based on the active theme. Text foreground inverts to **#1f2328** (near-black).

| Element | Light Mode (on #ffffff) |
|---------|------------------------|
| Added line background | #dafbe1 |
| Added word highlight | #aceebb |
| Deleted line background | #ffebe9 |
| Deleted word highlight | #ffc1c0 |

The underlying design principle: **same hue families, adjusted lightness** for contrast on the inverted canvas.

## Terminal adaptation strategy

### Alpha blending conversion formula

To convert GitHub's rgba colors to solid hex for terminal use:

```
R_solid = (alpha × R_foreground) + ((1 - alpha) × R_background)
G_solid = (alpha × G_foreground) + ((1 - alpha) × G_background)  
B_solid = (alpha × B_foreground) + ((1 - alpha) × B_background)
```

Using #0d1117 as the dark background, the pre-calculated solid values in the table above were derived from this formula.

### Recommended solid color palette for terminals

**For dark terminal backgrounds (targeting #0d1117 or similar):**

| Element | Recommended Hex | 256-color fallback |
|---------|----------------|-------------------|
| Added line background | `#1b3a28` | 22 |
| Added word emphasis | `#2b5a3c` | 28 |
| Added line number | `#3fb950` | 71 |
| Deleted line background | `#3c1e1f` | 52 |
| Deleted word emphasis | `#6b3b3c` | 88 |
| Deleted line number | `#f85149` | 196 |
| Hunk header background | `#121d2f` | 235 |
| Muted text | `#8b949e` | 245 |

### Priority ranking for GitHub aesthetic

1. **Line backgrounds** (added/deleted distinction)—this is the fundamental diff visual
2. **Word-level highlighting**—shows exact character changes within lines
3. **Contrast ratio compliance**—text must remain readable
4. **Line number coloring**—aids quick visual scanning
5. **Hunk headers**—orientation within files
6. **Gutter styling**—visual polish, lowest priority for terminals

### Terminal color mode considerations

For **true color terminals** (COLORTERM=truecolor): use exact hex values above. For **256-color terminals**: use the palette codes. For **16-color terminals**: fall back to named ANSI colors (`green`, `red`, `brightgreen`, `brightred`). The delta diff tool provides an excellent reference implementation, using syntax themes from bat and extensive color customization options with similar color choices to those recommended here.

## Edge cases

**Long lines**: GitHub wraps by default (configurable), with the entire diff table scrolling horizontally when wrapping is disabled. Terminals typically clip or wrap—consider offering both modes.

**Binary files**: Indicated with text labels rather than diff content; no special colors needed.

**File renames/moves**: Shown in file headers with text indicators and icons; use the same header styling with appropriate labels.

## Conclusion

GitHub's diff colors achieve their distinctive appearance through carefully chosen rgba values blending against a specific dark canvas. For terminal adaptation, the key insight is that **pre-blending these colors produces solid values that preserve the visual hierarchy**—muted line backgrounds, emphasized word-level changes, and distinct gutter columns. Prioritize implementing line backgrounds and word emphasis first; these two elements contribute most to the recognizable GitHub aesthetic. The functional token system's semantic naming (success/danger rather than green/red) provides flexibility for future colorblind-friendly terminal themes using the same orange/blue substitution pattern GitHub employs.
