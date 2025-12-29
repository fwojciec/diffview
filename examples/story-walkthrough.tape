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
Set Width 1400
Set Height 800
Set Padding 20
Set TypingSpeed 100ms

# Build and run replay
Type "go run ./cmd/diffstory replay eval-curated.jsonl 0"
Enter
Sleep 2s

# On intro slide - show summary
Sleep 1500ms

# Navigate through sections
Type "s"
Sleep 1500ms

Type "s"
Sleep 1500ms

Type "s"
Sleep 1500ms

# Scroll down within section
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 500ms

# Toggle collapsed hunks
Type "z"
Sleep 1s

# Scroll more
Type "j"
Sleep 200ms
Type "j"
Sleep 200ms
Type "j"
Sleep 500ms

# Go to previous section
Type "S"
Sleep 1s

# Scroll to top
Type "g"
Type "g"
Sleep 500ms

# Quit
Type "q"
Sleep 500ms
