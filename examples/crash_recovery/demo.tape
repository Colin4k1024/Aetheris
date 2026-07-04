# Aetheris Crash Recovery Demo — VHS tape
# https://github.com/charmbracelet/vhs
#
# Usage:
#   vhs demo.tape
#   # produces crash-recovery-demo.gif in the same directory

Output crash-recovery-demo.gif

Set Width  900
Set Height 600
Set Theme  Charm
Set FontSize 16
Set TypingSpeed 0ms
Set PlaybackSpeed 1.0

# Give the terminal a moment to initialize
Sleep 500ms

Type "python3 demo.py --no-color"
Enter

# Wait for the demo to finish (~25 seconds)
Sleep 28000ms
