#!/bin/bash
INPUT=$(cat)

LAST_MESSAGE=$(echo "$INPUT" | jq -r '.last_assistant_message // empty')
STOP_ACTIVE=$(echo "$INPUT" | jq -r '.stop_hook_active')
CWD=$(echo "$INPUT" | jq -r '.cwd // empty')

# Prevent infinite loops
if [ "$STOP_ACTIVE" = "true" ]; then
  exit 0
fi

if [ -z "$SLACK_WEBHOOK_URL" ] || [ -z "$LAST_MESSAGE" ]; then
  exit 0
fi

# Repo name from git remote or folder name
REPO=$(cd "$CWD" 2>/dev/null && git remote get-url origin 2>/dev/null | sed 's/.*[:/]\([^/]*\/[^/]*\)\.git$/\1/' || basename "$CWD")
BRANCH=$(cd "$CWD" 2>/dev/null && git branch --show-current 2>/dev/null || echo "unknown")

# Truncate message to 3000 chars to stay within Slack limits
TRUNCATED="${LAST_MESSAGE:0:3000}"
if [ ${#LAST_MESSAGE} -gt 3000 ]; then
  TRUNCATED="${TRUNCATED}... (truncated)"
fi

# Build the full message
FULL_MESSAGE="────────────────────────────────────────
Repository: ${REPO}
Branch: ${BRANCH}

${TRUNCATED}"

# Escape for JSON
ESCAPED=$(echo "$FULL_MESSAGE" | jq -Rs .)

curl -s -X POST "$SLACK_WEBHOOK_URL" \
  -H 'Content-type: application/json' \
  -d "{\"text\":${ESCAPED}}"

exit 0
