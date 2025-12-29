# VHS tape for a quick diffstory walkthrough (LinkedIn-friendly, <5MB)
#
# Usage:
#   vhs examples/story-quick.tape

Output examples/story-quick.gif

# Terminal configuration
Set Shell "bash"
Set FontSize 16
Set FontFamily "Iosevka Term"
Set Width 1400
Set Height 800
Set Padding 20
Set TypingSpeed 100ms

# Launch (hidden)
Hide
Type "go run ./cmd/diffstory replay eval-curated.jsonl 0"
Enter
Sleep 2s
Show

# Screen 1: Intro
Sleep 1500ms

# Screen 2: Defining Saver contract
Type "s"
Sleep 800ms
Type "jjj"
Sleep 600ms

# Screen 3: JSONL implementation
Type "s"
Sleep 800ms
Type "jjjjj"
Sleep 600ms

# Screen 4: Eval review integration
Type "s"
Sleep 800ms
Type "jjj"
Sleep 600ms

# Screen 5: Story viewer integration
Type "s"
Sleep 800ms

# Screen 6: CLI entrypoints wiring
Type "s"
Sleep 800ms

# Screen 7: Testing
Type "s"
Sleep 800ms
Type "jjjj"
Sleep 600ms

# Screen 8: Docs changes
Type "s"
Sleep 1s

# Quick return to intro
Type "SSSSSSS"
Sleep 1500ms

# Quit
Type "q"
Sleep 300ms
