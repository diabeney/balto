#!/bin/bash


HOOKS_DIR=".git/hooks"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

if [ ! -d ".git" ]; then
    echo "Error: .git directory not found. Please initialize git first with 'git init'"
    exit 1
fi

mkdir -p "$HOOKS_DIR"

cp "$PROJECT_ROOT/scripts/hooks/pre-commit" "$HOOKS_DIR/pre-commit"
chmod +x "$HOOKS_DIR/pre-commit"

echo "Git hooks installed successfully!"
echo "The pre-commit hook will now run 'go fmt' and 'golangci-lint' on every commit."

