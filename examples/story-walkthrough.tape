# VHS tape for recording diffstory story-guided review walkthrough
#
# This script demonstrates navigating through a classified diff using the
# story sections to understand the change structure.
#
# Usage:
#   vhs examples/story-walkthrough.tape
#
# Requirements:
#   - VHS (charmbracelet/vhs) installed: https://github.com/charmbracelet/vhs
#   - Go 1.21+ installed
#   - eval-curated.jsonl with at least one case

Output examples/story-walkthrough.gif

# Terminal configuration
Set Shell "bash"
Set FontSize 16
Set FontFamily "Iosevka Term"
Set Width 1400
Set Height 800
Set Padding 20
Set TypingSpeed 100ms

# Build and run replay (hidden - skip the typing)
Hide
Type "go run ./cmd/diffstory replay eval-curated.jsonl 0"
Enter
Sleep 2s
Show

# Screen 1: Intro - linger to show summary
Sleep 2s

# Screen 2: Defining Saver contract
Type "s"
Sleep 1500ms

# Screen 3: JSONL implementation - demo scrolling down and back up
Type "s"
Sleep 1s
Type "jjjjjjjjjjj"
Sleep 1s
Type "kkkkkkkkkkk"
Sleep 1s

# Screen 4: Eval review integration - demo gg and G
Type "s"
Sleep 1s
Type "G"
Sleep 1s
Type "gg"
Sleep 1s

# Screen 5: Story viewer integration - linger
Type "s"
Sleep 1500ms

# Screen 6: CLI entrypoints wiring - linger
Type "s"
Sleep 1500ms

# Screen 7: Testing - jump to bottom, unfold, scroll, fold back
Type "s"
Sleep 1s
Type "G"
Sleep 800ms
Type "z"
Sleep 1s
Type "jjjjjjjjjj"
Sleep 1s
Type "z"
Sleep 1s

# Screen 8: Docs changes - linger
Type "s"
Sleep 1500ms

# Navigate back through all sections with short pauses
Type "S"
Sleep 600ms
Type "S"
Sleep 600ms
Type "S"
Sleep 600ms
Type "S"
Sleep 600ms
Type "S"
Sleep 600ms
Type "S"
Sleep 600ms
Type "S"
Sleep 3s

# Quit
Type "q"
Sleep 500ms
