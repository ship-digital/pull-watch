#!/bin/bash
set -e

# Get the help output
HELP_OUTPUT=$(go run ./cmd/pull-watch -h | sed 's/^/  /')

# Create temp file
tmp=$(mktemp)

# Update README.md with new usage
awk -v help="$HELP_OUTPUT" '
BEGIN { in_usage = 0 }
/^## 🎮 Usage$/ {
    print
    print ""
    print "```"
    print help
    print "```"
    in_usage = 1
    next
}
/^##/ { in_usage = 0 }  # Reset when we hit next section
!in_usage { print }
' README.md > "$tmp"

mv "$tmp" README.md